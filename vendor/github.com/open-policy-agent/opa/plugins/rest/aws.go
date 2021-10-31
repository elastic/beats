// Copyright 2019 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package rest

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/open-policy-agent/opa/logging"
)

const (
	// ref. https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html
	ec2DefaultCredServicePath = "http://169.254.169.254/latest/meta-data/iam/security-credentials/"

	// ref. https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/configuring-instance-metadata-service.html
	ec2DefaultTokenPath = "http://169.254.169.254/latest/api/token"

	// ref. https://docs.aws.amazon.com/AmazonECS/latest/userguide/task-iam-roles.html
	ecsDefaultCredServicePath = "http://169.254.170.2"
	ecsRelativePathEnvVar     = "AWS_CONTAINER_CREDENTIALS_RELATIVE_URI"

	// ref. https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_enable-regions.html
	stsDefaultPath = "https://sts.amazonaws.com"
	stsRegionPath  = "https://sts.%s.amazonaws.com"

	// ref. https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html
	accessKeyEnvVar               = "AWS_ACCESS_KEY_ID"
	secretKeyEnvVar               = "AWS_SECRET_ACCESS_KEY"
	securityTokenEnvVar           = "AWS_SECURITY_TOKEN"
	sessionTokenEnvVar            = "AWS_SESSION_TOKEN"
	awsRegionEnvVar               = "AWS_REGION"
	awsRoleArnEnvVar              = "AWS_ROLE_ARN"
	awsWebIdentityTokenFileEnvVar = "AWS_WEB_IDENTITY_TOKEN_FILE"
)

// Headers that may be mutated before reaching an aws service (eg by a proxy) should be added here to omit them from
// the sigv4 canonical request
// ref. https://github.com/aws/aws-sdk-go/blob/master/aws/signer/v4/v4.go#L92
var awsSigv4IgnoredHeaders = map[string]bool{
	"authorization":   true,
	"user-agent":      true,
	"x-amzn-trace-id": true,
}

// awsCredentials represents the credentials obtained from an AWS credential provider
type awsCredentials struct {
	AccessKey    string
	SecretKey    string
	RegionName   string
	SessionToken string
}

// awsCredentialService represents the interface for AWS credential providers
type awsCredentialService interface {
	credentials() (awsCredentials, error)
}

// awsEnvironmentCredentialService represents an static environment-variable credential provider for AWS
type awsEnvironmentCredentialService struct {
	logger logging.Logger
}

func (cs *awsEnvironmentCredentialService) credentials() (awsCredentials, error) {
	var creds awsCredentials
	creds.AccessKey = os.Getenv(accessKeyEnvVar)
	if creds.AccessKey == "" {
		return creds, errors.New("no " + accessKeyEnvVar + " set in environment")
	}
	creds.SecretKey = os.Getenv(secretKeyEnvVar)
	if creds.SecretKey == "" {
		return creds, errors.New("no " + secretKeyEnvVar + " set in environment")
	}
	creds.RegionName = os.Getenv(awsRegionEnvVar)
	if creds.RegionName == "" {
		return creds, errors.New("no " + awsRegionEnvVar + " set in environment")
	}
	// SessionToken is required if using temporaty ENV credentials from assumed IAM role
	// Missing SessionToken results with 403 s3 error.
	creds.SessionToken = os.Getenv(sessionTokenEnvVar)
	if creds.SessionToken == "" {
		// In case of missing SessionToken try to get SecurityToken
		// AWS switched to use SessionToken, but SecurityToken was left for backward compatibility
		creds.SessionToken = os.Getenv(securityTokenEnvVar)
	}

	return creds, nil
}

// awsMetadataCredentialService represents an EC2 metadata service credential provider for AWS
type awsMetadataCredentialService struct {
	RoleName        string `json:"iam_role,omitempty"`
	RegionName      string `json:"aws_region"`
	creds           awsCredentials
	expiration      time.Time
	credServicePath string
	tokenPath       string
	logger          logging.Logger
}

func (cs *awsMetadataCredentialService) urlForMetadataService() (string, error) {
	// override default path for testing
	if cs.credServicePath != "" {
		return cs.credServicePath + cs.RoleName, nil
	}
	// otherwise, normal flow
	// if a role name is provided, look up via the EC2 credential service
	if cs.RoleName != "" {
		return ec2DefaultCredServicePath + cs.RoleName, nil
	}
	// otherwise, check environment to see if it looks like we're in an ECS
	// container (with implied role association)
	if isECS() {
		return ecsDefaultCredServicePath + os.Getenv(ecsRelativePathEnvVar), nil
	}
	// if there's no role name and we don't appear to have a path to the
	// ECS container service, then the configuration is invalid
	return "", errors.New("metadata endpoint cannot be determined from settings and environment")
}

func (cs *awsMetadataCredentialService) tokenRequest() (*http.Request, error) {
	tokenURL := ec2DefaultTokenPath
	if cs.tokenPath != "" {
		// override for testing
		tokenURL = cs.tokenPath
	}
	req, err := http.NewRequest(http.MethodPut, tokenURL, nil)
	if err != nil {
		return nil, err
	}

	// we are going to use the token in the immediate future, so a long TTL is not necessary
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "60")
	return req, nil
}

func (cs *awsMetadataCredentialService) refreshFromService() error {
	// define the expected JSON payload from the EC2 credential service
	// ref. https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html
	type metadataPayload struct {
		Code            string
		AccessKeyID     string `json:"AccessKeyId"`
		SecretAccessKey string
		Token           string
		Expiration      time.Time
	}

	// short circuit if a reasonable amount of time until credential expiration remains
	if time.Now().Add(time.Minute * 5).Before(cs.expiration) {
		cs.logger.Debug("Credentials previously obtained from metadata service still valid.")
		return nil
	}

	cs.logger.Debug("Obtaining credentials from metadata service.")
	metaDataURL, err := cs.urlForMetadataService()
	if err != nil {
		// configuration issue or missing ECS environment
		return err
	}

	// construct an HTTP client with a reasonably short timeout
	client := &http.Client{Timeout: time.Second * 10}
	req, err := http.NewRequest(http.MethodGet, metaDataURL, nil)
	if err != nil {
		return errors.New("unable to construct metadata HTTP request: " + err.Error())
	}

	// if in the EC2 environment, we will use IMDSv2, which requires a session cookie from a
	// PUT request on the token endpoint before it will give the credentials, this provides
	// protection from SSRF attacks
	if !isECS() {
		tokenReq, err := cs.tokenRequest()
		if err != nil {
			return errors.New("unable to construct metadata token HTTP request: " + err.Error())
		}
		body, err := doMetaDataRequestWithClient(tokenReq, client, "metadata token", cs.logger)
		if err != nil {
			return err
		}
		// token is the body of response; add to header of metadata request
		req.Header.Set("X-aws-ec2-metadata-token", string(body))
	}

	body, err := doMetaDataRequestWithClient(req, client, "metadata", cs.logger)
	if err != nil {
		return err
	}

	var payload metadataPayload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		return errors.New("failed to parse credential response from metadata service: " + err.Error())
	}

	// Only the EC2 endpoint returns the "Code" element which indicates whether the query was
	// successful; the ECS endpoint does not! Some other fields are missing in the ECS payload
	// but we do not depend on them.
	if cs.RoleName != "" && payload.Code != "Success" {
		return errors.New("metadata service query did not succeed: " + payload.Code)
	}

	cs.expiration = payload.Expiration
	cs.creds.AccessKey = payload.AccessKeyID
	cs.creds.SecretKey = payload.SecretAccessKey
	cs.creds.SessionToken = payload.Token
	cs.creds.RegionName = cs.RegionName

	return nil
}

func (cs *awsMetadataCredentialService) credentials() (awsCredentials, error) {
	err := cs.refreshFromService()
	if err != nil {
		return cs.creds, err
	}
	return cs.creds, nil
}

// awsWebIdentityCredentialService represents an STS WebIdentity credential services
type awsWebIdentityCredentialService struct {
	RoleArn              string
	WebIdentityTokenFile string
	RegionName           string `json:"aws_region"`
	SessionName          string `json:"session_name"`
	stsURL               string
	creds                awsCredentials
	expiration           time.Time
	logger               logging.Logger
}

func (cs *awsWebIdentityCredentialService) populateFromEnv() error {
	cs.RoleArn = os.Getenv(awsRoleArnEnvVar)
	if cs.RoleArn == "" {
		return errors.New("no " + awsRoleArnEnvVar + " set in environment")
	}
	cs.WebIdentityTokenFile = os.Getenv(awsWebIdentityTokenFileEnvVar)
	if cs.WebIdentityTokenFile == "" {
		return errors.New("no " + awsWebIdentityTokenFileEnvVar + " set in environment")
	}

	if cs.RegionName == "" {
		if cs.RegionName = os.Getenv(awsRegionEnvVar); cs.RegionName == "" {
			return errors.New("no " + awsRegionEnvVar + " set in environment or configuration")
		}
	}
	return nil
}

func (cs *awsWebIdentityCredentialService) stsPath() string {
	var stsPath string
	switch {
	case cs.stsURL != "":
		stsPath = cs.stsURL
	case cs.RegionName != "":
		stsPath = fmt.Sprintf(stsRegionPath, strings.ToLower(cs.RegionName))
	default:
		stsPath = stsDefaultPath
	}
	return stsPath
}

func (cs *awsWebIdentityCredentialService) refreshFromService() error {
	// define the expected JSON payload from the EC2 credential service
	// ref. https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRoleWithWebIdentity.html
	type responsePayload struct {
		Result struct {
			Credentials struct {
				SessionToken    string
				SecretAccessKey string
				Expiration      time.Time
				AccessKeyID     string `xml:"AccessKeyId"`
			}
		} `xml:"AssumeRoleWithWebIdentityResult"`
	}

	// short circuit if a reasonable amount of time until credential expiration remains
	if time.Now().Add(time.Minute * 5).Before(cs.expiration) {
		cs.logger.Debug("Credentials previously obtained from sts service still valid.")
		return nil
	}

	cs.logger.Debug("Obtaining credentials from sts for role %s.", cs.RoleArn)

	var sessionName string
	if cs.SessionName == "" {
		sessionName = "open-policy-agent"
	} else {
		sessionName = cs.SessionName
	}

	tokenData, err := ioutil.ReadFile(cs.WebIdentityTokenFile)
	if err != nil {
		return errors.New("unable to read web token for sts HTTP request: " + err.Error())
	}

	token := string(tokenData)

	queryVals := url.Values{
		"Action":           []string{"AssumeRoleWithWebIdentity"},
		"RoleSessionName":  []string{sessionName},
		"RoleArn":          []string{cs.RoleArn},
		"WebIdentityToken": []string{token},
		"Version":          []string{"2011-06-15"},
	}
	stsRequestURL, _ := url.Parse(cs.stsPath())

	// construct an HTTP client with a reasonably short timeout
	client := &http.Client{Timeout: time.Second * 10}
	req, err := http.NewRequest(http.MethodPost, stsRequestURL.String(), strings.NewReader(queryVals.Encode()))
	if err != nil {
		return errors.New("unable to construct STS HTTP request: " + err.Error())
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	body, err := doMetaDataRequestWithClient(req, client, "STS", cs.logger)
	if err != nil {
		return err
	}

	var payload responsePayload
	err = xml.Unmarshal(body, &payload)
	if err != nil {
		return errors.New("failed to parse credential response from STS service: " + err.Error())
	}

	cs.expiration = payload.Result.Credentials.Expiration
	cs.creds.AccessKey = payload.Result.Credentials.AccessKeyID
	cs.creds.SecretKey = payload.Result.Credentials.SecretAccessKey
	cs.creds.SessionToken = payload.Result.Credentials.SessionToken
	cs.creds.RegionName = cs.RegionName

	return nil
}

func (cs *awsWebIdentityCredentialService) credentials() (awsCredentials, error) {
	err := cs.refreshFromService()
	if err != nil {
		return cs.creds, err
	}
	return cs.creds, nil
}

func isECS() bool {
	// the special relative path URI is set by the container agent in the ECS environment only
	_, isECS := os.LookupEnv(ecsRelativePathEnvVar)
	return isECS
}

func doMetaDataRequestWithClient(req *http.Request, client *http.Client, desc string, logger logging.Logger) ([]byte, error) {
	// convenience function to get the body of an AWS EC2 metadata service request with
	// appropriate error-handling boilerplate and logging for this special case
	resp, err := client.Do(req)
	if err != nil {
		// some kind of catastrophe talking to the EC2 service
		return nil, errors.New(desc + " HTTP request failed: " + err.Error())
	}
	defer resp.Body.Close()

	logger.WithFields(map[string]interface{}{
		"url":     req.URL.String(),
		"status":  resp.Status,
		"headers": resp.Header,
	}).Debug("Received response from " + desc + " service.")

	if resp.StatusCode != 200 {
		// could be 404 for role that's not available, but cover all the bases
		return nil, errors.New(desc + " HTTP request returned unexpected status: " + resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// deal with problems reading the body, whatever those might be
		return nil, errors.New(desc + " HTTP response body could not be read: " + err.Error())
	}
	return body, nil
}

func sha256MAC(message []byte, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

func sortKeys(strMap map[string][]string) []string {
	keys := make([]string, len(strMap))

	i := 0
	for k := range strMap {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

// signV4 modifies an http.Request to include an AWS V4 signature based on a credential provider
func signV4(req *http.Request, service string, credService awsCredentialService, theTime time.Time) error {
	// General ref. https://docs.aws.amazon.com/general/latest/gr/sigv4_signing.html
	// S3 ref. https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-auth-using-authorization-header.html
	// APIGateway ref. https://docs.aws.amazon.com/apigateway/api-reference/signing-requests/

	var body []byte
	if req.Body == nil {
		body = []byte("")
	} else {
		var err error
		body, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return errors.New("error getting request body: " + err.Error())
		}
		// Since ReadAll consumed the body ReadCloser, we must create a new ReadCloser for the request so that the
		// subsequent read starts from the beginning
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
	}
	creds, err := credService.credentials()
	if err != nil {
		return errors.New("error getting AWS credentials: " + err.Error())
	}

	bodyHexHash := fmt.Sprintf("%x", sha256.Sum256(body))

	now := theTime.UTC()

	// V4 signing has specific ideas of how it wants to see dates/times encoded
	dateNow := now.Format("20060102")
	iso8601Now := now.Format("20060102T150405Z")

	awsHeaders := map[string]string{
		"host":       req.URL.Host,
		"x-amz-date": iso8601Now,
	}

	// s3 and glacier require the extra x-amz-content-sha256 header. other services do not.
	if service == "s3" || service == "glacier" {
		awsHeaders["x-amz-content-sha256"] = bodyHexHash
	}

	// the security token header is necessary for ephemeral credentials, e.g. from
	// the EC2 metadata service
	if creds.SessionToken != "" {
		awsHeaders["x-amz-security-token"] = creds.SessionToken
	}

	headersToSign := map[string][]string{}

	// sign all of the aws headers
	for k, v := range awsHeaders {
		headersToSign[k] = []string{v}
	}

	// sign all of the request's headers, except for those in the ignore list
	for k, v := range req.Header {
		lowerCaseHeader := strings.ToLower(k)
		if !awsSigv4IgnoredHeaders[lowerCaseHeader] {
			headersToSign[lowerCaseHeader] = v
		}
	}

	// the "canonical request" is the normalized version of the AWS service access
	// that we're attempting to perform; in this case, a GET from an S3 bucket
	canonicalReq := req.Method + "\n"            // HTTP method
	canonicalReq += req.URL.EscapedPath() + "\n" // URI-escaped path
	canonicalReq += "\n"                         // query string; not implemented

	// include the values for the signed headers
	orderedKeys := sortKeys(headersToSign)
	for _, k := range orderedKeys {
		canonicalReq += k + ":" + strings.Join(headersToSign[k], ",") + "\n"
	}
	canonicalReq += "\n" // linefeed to terminate headers

	// include the list of the signed headers
	headerList := strings.Join(orderedKeys, ";")
	canonicalReq += headerList + "\n"
	canonicalReq += bodyHexHash

	// the "string to sign" is a time-bounded, scoped request token which
	// is linked to the "canonical request" by inclusion of its SHA-256 hash
	strToSign := "AWS4-HMAC-SHA256\n"                                                 // V4 signing with SHA-256 HMAC
	strToSign += iso8601Now + "\n"                                                    // ISO 8601 time
	strToSign += dateNow + "/" + creds.RegionName + "/" + service + "/aws4_request\n" // scoping for signature
	strToSign += fmt.Sprintf("%x", sha256.Sum256([]byte(canonicalReq)))               // SHA-256 of canonical request

	// the "signing key" is generated by repeated HMAC-SHA256 based on the same
	// scoping that's included in the "string to sign"; but including the secret key
	// to allow AWS to validate it
	signingKey := sha256MAC([]byte(dateNow), []byte("AWS4"+creds.SecretKey))
	signingKey = sha256MAC([]byte(creds.RegionName), signingKey)
	signingKey = sha256MAC([]byte(service), signingKey)
	signingKey = sha256MAC([]byte("aws4_request"), signingKey)

	// the "signature" is finally the "string to sign" signed by the "signing key"
	signature := sha256MAC([]byte(strToSign), signingKey)

	// required format of Authorization header; n.b. the access key corresponding to
	// the secret key is included here
	authHdr := "AWS4-HMAC-SHA256 Credential=" + creds.AccessKey + "/" + dateNow
	authHdr += "/" + creds.RegionName + "/" + service + "/aws4_request,"
	authHdr += "SignedHeaders=" + headerList + ","
	authHdr += "Signature=" + fmt.Sprintf("%x", signature)

	// add the computed Authorization
	req.Header.Set("Authorization", authHdr)

	// populate the other signed headers into the request
	for k := range awsHeaders {
		req.Header.Add(k, awsHeaders[k])
	}

	return nil
}
