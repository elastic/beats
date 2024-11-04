package usage

type UsageResponse struct {
	Object                       string        `json:"object"`
	Data                         []UsageData   `json:"data"`
	FtData                       []interface{} `json:"ft_data"`
	DalleApiData                 []DalleData   `json:"dalle_api_data"`
	WhisperApiData               []WhisperData `json:"whisper_api_data"`
	TtsApiData                   []TtsData     `json:"tts_api_data"`
	AssistantCodeInterpreterData []interface{} `json:"assistant_code_interpreter_data"`
	RetrievalStorageData         []interface{} `json:"retrieval_storage_data"`
}

type UsageData struct {
	OrganizationID            string  `json:"organization_id"`
	OrganizationName          string  `json:"organization_name"`
	AggregationTimestamp      int64   `json:"aggregation_timestamp"`
	NRequests                 int     `json:"n_requests"`
	Operation                 string  `json:"operation"`
	SnapshotID                string  `json:"snapshot_id"`
	NContextTokensTotal       int     `json:"n_context_tokens_total"`
	NGeneratedTokensTotal     int     `json:"n_generated_tokens_total"`
	Email                     *string `json:"email"`
	ApiKeyID                  *string `json:"api_key_id"`
	ApiKeyName                *string `json:"api_key_name"`
	ApiKeyRedacted            *string `json:"api_key_redacted"`
	ApiKeyType                *string `json:"api_key_type"`
	ProjectID                 *string `json:"project_id"`
	ProjectName               *string `json:"project_name"`
	RequestType               string  `json:"request_type"`
	NCachedContextTokensTotal int     `json:"n_cached_context_tokens_total"`
}

type DalleData struct {
	Timestamp        int64   `json:"timestamp"`
	NumImages        int     `json:"num_images"`
	NumRequests      int     `json:"num_requests"`
	ImageSize        string  `json:"image_size"`
	Operation        string  `json:"operation"`
	UserID           *string `json:"user_id"`
	OrganizationID   string  `json:"organization_id"`
	ApiKeyID         *string `json:"api_key_id"`
	ApiKeyName       *string `json:"api_key_name"`
	ApiKeyRedacted   *string `json:"api_key_redacted"`
	ApiKeyType       *string `json:"api_key_type"`
	OrganizationName string  `json:"organization_name"`
	ModelID          string  `json:"model_id"`
	ProjectID        *string `json:"project_id"`
	ProjectName      *string `json:"project_name"`
}

type WhisperData struct {
	Timestamp        int64   `json:"timestamp"`
	ModelID          string  `json:"model_id"`
	NumSeconds       int     `json:"num_seconds"`
	NumRequests      int     `json:"num_requests"`
	UserID           *string `json:"user_id"`
	OrganizationID   string  `json:"organization_id"`
	ApiKeyID         *string `json:"api_key_id"`
	ApiKeyName       *string `json:"api_key_name"`
	ApiKeyRedacted   *string `json:"api_key_redacted"`
	ApiKeyType       *string `json:"api_key_type"`
	OrganizationName string  `json:"organization_name"`
	ProjectID        *string `json:"project_id"`
	ProjectName      *string `json:"project_name"`
}

type TtsData struct {
	Timestamp        int64   `json:"timestamp"`
	ModelID          string  `json:"model_id"`
	NumCharacters    int     `json:"num_characters"`
	NumRequests      int     `json:"num_requests"`
	UserID           *string `json:"user_id"`
	OrganizationID   string  `json:"organization_id"`
	ApiKeyID         *string `json:"api_key_id"`
	ApiKeyName       *string `json:"api_key_name"`
	ApiKeyRedacted   *string `json:"api_key_redacted"`
	ApiKeyType       *string `json:"api_key_type"`
	OrganizationName string  `json:"organization_name"`
	ProjectID        *string `json:"project_id"`
	ProjectName      *string `json:"project_name"`
}
