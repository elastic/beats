package kafka

import "time"

const (
	ActivityLogs         = "ActivityLogs"
	AuditLogs            = "AuditLogs"
)

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

type AzureActivityLogs struct {
	Records []ActivityLog `json:"records"`
}
