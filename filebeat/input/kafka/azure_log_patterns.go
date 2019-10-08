// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package kafka

import "time"

// AzureActivityLogs structure matches the eventhub message carrying the azure activity logs
type AzureActivityLogs struct {
	Records []ActivityLog `json:"records"`
}

// AzureAuditLogs structure matches the eventhub message carrying the azure audit logs
type AzureAuditLogs struct {
	Records []AuditLog `json:"records"`
}

// AzureSigninLogs structure matches the eventhub message carrying the azure signin logs
type AzureSigninLogs struct {
	Records []SigninLog `json:"records"`
}

// ActivityLogs structure matches the azure activity log format
type ActivityLog struct {
	Time            time.Time `json:"time"`
	ResourceID      string    `json:"resourceId"`
	OperationName   string    `json:"operationName"`
	Category        string    `json:"category"`
	ResultType      string    `json:"resultType"`
	ResultSignature string    `json:"resultSignature"`
	DurationMs      int       `json:"durationMs"`
	CallerIPAddress string    `json:"callerIpAddress"`
	CorrelationID   string    `json:"correlationId"`
	Identity        struct {
		Authorization struct {
			Scope    string `json:"scope"`
			Action   string `json:"action"`
			Evidence struct {
				Role                string `json:"role"`
				RoleAssignmentScope string `json:"roleAssignmentScope"`
				RoleAssignmentID    string `json:"roleAssignmentId"`
				RoleDefinitionID    string `json:"roleDefinitionId"`
				PrincipalID         string `json:"principalId"`
				PrincipalType       string `json:"principalType"`
			} `json:"evidence"`
		} `json:"authorization"`
		Claims struct {
			Aud                                                       string `json:"aud"`
			Iss                                                       string `json:"iss"`
			Iat                                                       string `json:"iat"`
			Nbf                                                       string `json:"nbf"`
			Exp                                                       string `json:"exp"`
			Aio                                                       string `json:"aio"`
			Appid                                                     string `json:"appid"`
			Appidacr                                                  string `json:"appidacr"`
			HTTPSchemasMicrosoftComIdentityClaimsIdentityprovider     string `json:"http://schemas.microsoft.com/identity/claims/identityprovider"`
			HTTPSchemasMicrosoftComIdentityClaimsObjectidentifier     string `json:"http://schemas.microsoft.com/identity/claims/objectidentifier"`
			HTTPSchemasXmlsoapOrgWs200505IdentityClaimsNameidentifier string `json:"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/nameidentifier"`
			HTTPSchemasMicrosoftComIdentityClaimsTenantid             string `json:"http://schemas.microsoft.com/identity/claims/tenantid"`
			Uti                                                       string `json:"uti"`
			Ver                                                       string `json:"ver"`
		} `json:"claims"`
	} `json:"identity"`
	Level      string `json:"level"`
	Location   string `json:"location"`
	Properties struct {
		StatusCode       string      `json:"statusCode"`
		ServiceRequestID interface{} `json:"serviceRequestId"`
		StatusMessage    string      `json:"statusMessage"`
	} `json:"properties,omitempty"`
}

// AuditLog structure matches the azure audit log format
type AuditLog struct {
	Category         string `json:"category"`
	CorrelationID    string `json:"correlationId"`
	DurationMs       int    `json:"durationMs"`
	Level            string `json:"level"`
	OperationName    string `json:"operationName"`
	OperationVersion string `json:"operationVersion"`
	Properties       struct {
		ActivityDateTime    string        `json:"activityDateTime"`
		ActivityDisplayName string        `json:"activityDisplayName"`
		AdditionalDetails   []interface{} `json:"additionalDetails"`
		Category            string        `json:"category"`
		CorrelationID       string        `json:"correlationId"`
		ID                  string        `json:"id"`
		InitiatedBy         struct {
			App struct {
				AppID                interface{} `json:"appId"`
				DisplayName          string      `json:"displayName"`
				ServicePrincipalID   string      `json:"servicePrincipalId"`
				ServicePrincipalName interface{} `json:"servicePrincipalName"`
			} `json:"app"`
			User struct {
				DisplayName       interface{} `json:"displayName"`
				ID                string      `json:"id"`
				IPAddress         interface{} `json:"ipAddress"`
				UserPrincipalName string      `json:"userPrincipalName"`
			} `json:"user"`
		} `json:"initiatedBy"`
		LoggedByService string `json:"loggedByService"`
		OperationType   string `json:"operationType"`
		Result          string `json:"result"`
		ResultReason    string `json:"resultReason"`
		TargetResources []struct {
			DisplayName        interface{} `json:"displayName"`
			ID                 string      `json:"id"`
			ModifiedProperties []struct {
				DisplayName string `json:"displayName"`
				NewValue    string `json:"newValue"`
				OldValue    string `json:"oldValue"`
			} `json:"modifiedProperties"`
			Type              string `json:"type"`
			UserPrincipalName string `json:"userPrincipalName"`
		} `json:"targetResources"`
	} `json:"properties"`
	ResourceID      string `json:"resourceId"`
	ResultSignature string `json:"resultSignature"`
	TenantID        string `json:"tenantId"`
	Time            string `json:"time"`
	Identity        string `json:"identity"`
}

// SigninLog structure matches the azure audit log format
type SigninLog struct {
	Level            int    `json:"Level"`
	CallerIPAddress  string `json:"callerIpAddress"`
	Category         string `json:"category"`
	CorrelationID    string `json:"correlationId"`
	DurationMs       int    `json:"durationMs"`
	Identity         string `json:"identity"`
	Location         string `json:"location"`
	OperationName    string `json:"operationName"`
	OperationVersion string `json:"operationVersion"`
	Properties       struct {
		SignInBondData struct {
			ConditionalAccessDetails interface{} `json:"ConditionalAccessDetails"`
			DeviceDetails            struct {
				BrowserID         interface{} `json:"BrowserId"`
				BrowserType       string      `json:"BrowserType"`
				DeviceDisplayName string      `json:"DeviceDisplayName"`
				DeviceID          string      `json:"DeviceId"`
				DevicePlatform    string      `json:"DevicePlatform"`
				DeviceTrustType   int         `json:"DeviceTrustType"`
				IsCompliant       interface{} `json:"IsCompliant"`
				IsManaged         interface{} `json:"IsManaged"`
				UserAgent         string      `json:"UserAgent"`
			} `json:"DeviceDetails"`
			DisplayDetails struct {
				ApplicationDisplayName           string      `json:"ApplicationDisplayName"`
				AttemptedUsername                interface{} `json:"AttemptedUsername"`
				ProxyRestrictionTargetTenantName interface{} `json:"ProxyRestrictionTargetTenantName"`
				ResourceDisplayName              string      `json:"ResourceDisplayName"`
				UserName                         string      `json:"UserName"`
			} `json:"DisplayDetails"`
			IssuerDetails   interface{} `json:"IssuerDetails"`
			LocationDetails struct {
				IPChain   interface{} `json:"IPChain"`
				Latitude  float64     `json:"Latitude"`
				Longitude float64     `json:"Longitude"`
			} `json:"LocationDetails"`
			MfaDetails struct {
				AuthMethod     interface{} `json:"AuthMethod"`
				MaskedDeviceID interface{} `json:"MaskedDeviceId"`
				MfaStatus      int         `json:"MfaStatus"`
				SasStatus      interface{} `json:"SasStatus"`
			} `json:"MfaDetails"`
			PassThroughAuthenticationDetails interface{} `json:"PassThroughAuthenticationDetails"`
			ProtocolDetails                  struct {
				AuthenticationMethodsUsed interface{} `json:"AuthenticationMethodsUsed"`
				DomainHintPresent         interface{} `json:"DomainHintPresent"`
				IsInteractive             interface{} `json:"IsInteractive"`
				LoginHintPresent          interface{} `json:"LoginHintPresent"`
				NetworkLocation           interface{} `json:"NetworkLocation"`
				Protocol                  interface{} `json:"Protocol"`
				ResponseTime              int         `json:"ResponseTime"`
			} `json:"ProtocolDetails"`
			RAMDetails      interface{} `json:"RamDetails"`
			SourceAlpEvents interface{} `json:"SourceAlpEvents"`
		} `json:"SignInBondData"`
		AppDisplayName                   string        `json:"appDisplayName"`
		AppID                            string        `json:"appId"`
		AppliedConditionalAccessPolicies []interface{} `json:"appliedConditionalAccessPolicies"`
		AuthenticationDetails            []struct {
			AuthenticationStepDateTime     string `json:"authenticationStepDateTime"`
			AuthenticationStepRequirement  string `json:"authenticationStepRequirement"`
			AuthenticationStepResultDetail string `json:"authenticationStepResultDetail"`
			Succeeded                      bool   `json:"succeeded"`
		} `json:"authenticationDetails"`
		AuthenticationProcessingDetails   []interface{} `json:"authenticationProcessingDetails"`
		AuthenticationRequirementPolicies []interface{} `json:"authenticationRequirementPolicies"`
		ClientAppUsed                     string        `json:"clientAppUsed"`
		CorrelationID                     string        `json:"correlationId"`
		CreatedDateTime                   string        `json:"createdDateTime"`
		DeviceDetail                      struct {
			DeviceID        string `json:"deviceId"`
			DisplayName     string `json:"displayName"`
			OperatingSystem string `json:"operatingSystem"`
			TrustType       string `json:"trustType"`
		} `json:"deviceDetail"`
		ID            string `json:"id"`
		IPAddress     string `json:"ipAddress"`
		IsInteractive bool   `json:"isInteractive"`
		Location      struct {
			City            string `json:"city"`
			CountryOrRegion string `json:"countryOrRegion"`
			GeoCoordinates  struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"geoCoordinates"`
			State string `json:"state"`
		} `json:"location"`
		MfaDetail                    struct{}      `json:"mfaDetail"`
		NetworkLocationDetails       []interface{} `json:"networkLocationDetails"`
		OriginalRequestID            string        `json:"originalRequestId"`
		ProcessingTimeInMilliseconds int           `json:"processingTimeInMilliseconds"`
		ResourceDisplayName          string        `json:"resourceDisplayName"`
		ResourceID                   string        `json:"resourceId"`
		RiskDetail                   string        `json:"riskDetail"`
		RiskEventTypes               []interface{} `json:"riskEventTypes"`
		RiskLevelAggregated          string        `json:"riskLevelAggregated"`
		RiskLevelDuringSignIn        string        `json:"riskLevelDuringSignIn"`
		RiskState                    string        `json:"riskState"`
		Status                       struct {
			AdditionalDetails string `json:"additionalDetails"`
			ErrorCode         int    `json:"errorCode"`
		} `json:"status"`
		TokenIssuerName   string `json:"tokenIssuerName"`
		TokenIssuerType   string `json:"tokenIssuerType"`
		UserDisplayName   string `json:"userDisplayName"`
		UserID            string `json:"userId"`
		UserPrincipalName string `json:"userPrincipalName"`
	} `json:"properties"`
	ResourceID      string `json:"resourceId"`
	ResultSignature string `json:"resultSignature"`
	ResultType      string `json:"resultType"`
	TenantID        string `json:"tenantId"`
	Time            string `json:"time"`
}
