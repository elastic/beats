package wincrypt_client_certs

import (
	"crypto/sha1"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"os/user"
)

type X509Parameters x509.Certificate

func (certificate *X509Parameters)Get(name string)(interface{}, error) {
	// Callback for govaluate (https://github.com/Knetic/govaluate)
	// To provide variables for a x509 certificate

	switch name {
		case "X509_Version":
			return certificate.Version, nil
		case "X509_SerialNumber":
			return fmt.Sprintf("%x", certificate.SerialNumber), nil
		case "X509_SignatureAlgorithm":
			return certificate.SignatureAlgorithm.String(), nil
		case "X509_Issuer":
			return certificate.Issuer.String(), nil
		case "X509_Issuer_CommonName":
			return certificate.Issuer.CommonName, nil
		case "X509_Issuer_SerialNumber":
			return certificate.Issuer.SerialNumber, nil
		case "X509_Issuer_Country":
			return stringSliceTointerfaceSlice(certificate.Issuer.Country), nil
		case "X509_Issuer_Locality":
			return stringSliceTointerfaceSlice(certificate.Issuer.Locality), nil
		case "X509_Issuer_Organization":
			return stringSliceTointerfaceSlice(certificate.Issuer.Organization), nil
		case "X509_Issuer_OrganizationalUnit":
			return stringSliceTointerfaceSlice(certificate.Issuer.OrganizationalUnit), nil
		case "X509_Issuer_Province":
			return stringSliceTointerfaceSlice(certificate.Issuer.Province), nil
		case "X509_Issuer_StreetAddress":
			return stringSliceTointerfaceSlice(certificate.Issuer.StreetAddress), nil
		case "X509_Issuer_PostalCode":
			return stringSliceTointerfaceSlice(certificate.Issuer.PostalCode), nil
		case "X509_Subject":
			return certificate.Subject.String(), nil
		case "X509_Subject_CommonName":
			return certificate.Subject.CommonName, nil
		case "X509_Subject_SerialNumber":
			return certificate.Subject.SerialNumber, nil
		case "X509_Subject_Country":
			return stringSliceTointerfaceSlice(certificate.Subject.Country), nil
		case "X509_Subject_Locality":
			return stringSliceTointerfaceSlice(certificate.Subject.Locality), nil
		case "X509_Subject_Organization":
			return stringSliceTointerfaceSlice(certificate.Subject.Organization), nil
		case "X509_Subject_OrganizationalUnit":
			return stringSliceTointerfaceSlice(certificate.Subject.OrganizationalUnit), nil
		case "X509_Subject_Province":
			return stringSliceTointerfaceSlice(certificate.Subject.Province), nil
		case "X509_Subject_StreetAddress":
			return stringSliceTointerfaceSlice(certificate.Subject.StreetAddress), nil
		case "X509_Subject_PostalCode":
			return stringSliceTointerfaceSlice(certificate.Subject.PostalCode), nil
		case "X509_ValidFrom":
			return certificate.NotBefore, nil
		case "X509_ValidTo":
			return certificate.NotAfter, nil
		case "X509_PublicKeyAlgorithm":
			return certificate.PublicKeyAlgorithm.String(), nil
		case "X509_AuthorityKeyId":
			return fmt.Sprintf("%x", certificate.AuthorityKeyId), nil
		case "X509_SubjectKeyId":
			return fmt.Sprintf("%x", certificate.SubjectKeyId), nil
		case "X509_Fingerprint":
			sha1 := sha1.New()
			sha1.Write(certificate.Raw)
			return fmt.Sprintf("%x", sha1.Sum(nil)), nil
		case "X509_ExtKeyUsage":
			extKeyUsage := make([]interface{}, len(certificate.ExtKeyUsage))
			for i, u := range (certificate.ExtKeyUsage) {
				extKeyUsage[i] = extKeyUsageToString(u)
			}
			return extKeyUsage, nil

		case "Hostname":
			h, err := os.Hostname()
			if err != nil { return nil, errors.New("Failed to get information about the hostname") }
			return h, nil

		case "User_Username", "User_Name":
			u, err := user.Current()
			if err != nil { return nil, errors.New("Failed to get information about the current user") }

			if name == "User_Username" {
				return u.Username, nil
			} else {
				return u.Name, nil
			}
		default:
			return nil, fmt.Errorf("Unknown variable %s", name)
	}
}

func extKeyUsageToString(usage x509.ExtKeyUsage) string {
	switch(usage) {
		case x509.ExtKeyUsageServerAuth:
			return "ServerAuth"
		case x509.ExtKeyUsageClientAuth:
			return "ClientAuth"
		case x509.ExtKeyUsageCodeSigning:
			return "CodeSigning"
		case x509.ExtKeyUsageEmailProtection:
			return "EmailProtection"
		case x509.ExtKeyUsageIPSECEndSystem:
			return "IPSECEndSystem"
		case x509.ExtKeyUsageIPSECTunnel:
			return "IPSECTunnel"
		case x509.ExtKeyUsageIPSECUser:
			return "IPSECUser"
		case x509.ExtKeyUsageTimeStamping:
			return "TimeStamping"
		case x509.ExtKeyUsageOCSPSigning:
			return "OCSPSigning"
		case x509.ExtKeyUsageMicrosoftServerGatedCrypto:
			return "MicrosoftServerGatedCrypto"
		case x509.ExtKeyUsageNetscapeServerGatedCrypto:
			return "NetscapeServerGatedCrypto"
		case x509.ExtKeyUsageMicrosoftCommercialCodeSigning:
			return "MicrosoftCommercialCodeSigning"
		case x509.ExtKeyUsageMicrosoftKernelCodeSigning:
			return "MicrosoftKernelCodeSigning"
		default:
			return "Unknown"
	}
}

func stringSliceTointerfaceSlice(in []string) []interface{} {
	out := make([]interface{}, len(in))
	for i := 0; i < len(in); i++ {
		out[i] = in[i]
	}
	return out
}
