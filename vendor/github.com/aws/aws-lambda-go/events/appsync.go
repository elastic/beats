package events

import "encoding/json"

// AppSyncResolverTemplate represents the requests from AppSync to Lambda
type AppSyncResolverTemplate struct {
	Version   string           `json:"version"`
	Operation AppSyncOperation `json:"operation"`
	Payload   json.RawMessage  `json:"payload"`
}

// AppSyncOperation specifies the operation type supported by Lambda operations
type AppSyncOperation string

const (
	// OperationInvoke lets AWS AppSync know to call your Lambda function for every GraphQL field resolver
	OperationInvoke AppSyncOperation = "Invoke"
	// OperationBatchInvoke instructs AWS AppSync to batch requests for the current GraphQL field
	OperationBatchInvoke AppSyncOperation = "BatchInvoke"
)
