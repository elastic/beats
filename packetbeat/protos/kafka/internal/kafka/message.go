package kafka

type RawMessage struct {
	Payload []byte
}

type (
	RequestHeader struct {
		APIKey        APIKey
		Version       APIVersion
		CorrelationId ID
		ClientID      []byte
	}

	ResponseHeader struct {
		CorrelationID ID
	}

	MetadataRequest struct {
		Topics []string
	}

	MetadataResponse struct {
		Brokers []BrokerInfo
		Topics  []TopicMetadata
	}

	BrokerInfo struct {
		ID   ID
		Host string
		Port int32
	}

	TopicMetadata struct {
		Error      ErrorCode
		Name       string
		Partitions []PartitionMetadata
	}

	PartitionMetadata struct {
		Error    ErrorCode
		ID       ID
		Leader   ID
		Replicas []ID
		ISR      []ID
	}

	ProduceRequest struct {
		RequiredAcks int16
		Timeout      int32
		Topics       map[string]map[ID]RawMessageSet
	}

	ProduceResponse struct {
		ThrottleTime int32
		Topics       map[string]map[ID]ProduceResult
	}

	ProduceResult struct {
		Error     ErrorCode
		Offset    int64
		Timestamp int64
	}

	FetchRequest struct {
		ReplicaID   ID
		MaxWaitTime int32
		MinBytes    int32
		Topics      map[string]map[ID]FetchParams
	}

	FetchResponse struct {
		ThrottleTime int32
		Topics       map[string]map[ID]FetchResult
	}

	FetchParams struct {
		Offset   int64
		MaxBytes int32
	}

	FetchResult struct {
		Error      ErrorCode
		HWMOffset  int64
		MessageSet RawMessageSet
	}

	OffsetRequest struct {
		ReplicaID ID
		Topics    map[string]map[ID]OffsetParams
	}

	OffsetParams struct {
		Time       int64
		MaxOffsets int32
	}

	OffsetResponse struct {
		Topics map[string]map[ID]OffsetResult
	}

	OffsetResult struct {
		Error   ErrorCode
		Offsets []int64
	}

	GroupCoordinatorRequest struct {
		GroupID string
	}

	GroupCoordinatorResponse struct {
		Error           ErrorCode
		CoordinatorID   ID
		CoordinatorHost string
		CoordinatorPort int32
	}

	OffsetCommitRequest struct {
		GroupID           string
		GroupGenerationID ID
		ConsumerID        string
		RetentionTime     int64
		Topics            map[string]map[ID]CommitParams
	}

	CommitParams struct {
		Offset    int64
		Timestamp int64
		Metadata  []byte
	}

	OffsetCommitResponse struct {
		Topics map[string]map[ID]ErrorCode
	}

	OffsetFetchRequest struct {
		GroupID string
		Topics  map[string][]ID
	}

	OffsetFetchResponse struct {
		Topics map[string]map[ID]OffsetFetchResult
	}

	OffsetFetchResult struct {
		Offset   int64
		Metadata []byte
		Error    ErrorCode
	}

	JoinGroupRequest struct {
		GroupID          string
		SessionTimeout   int32
		RebalanceTimeout int32
		MemberID         string
		ProtocolType     string
		Protocols        map[string][]byte
	}

	JoinGroupResponse struct {
		Error         ErrorCode
		GenerationID  int32
		GroupProtocol string
		LeaderID      string
		MemberID      string
		Members       map[string][]byte
	}

	SyncGroupRequest struct {
		GroupID      string
		GenerationID int32
		MemberID     string
		Assignments  map[string][]byte
	}

	SyncGroupResponse struct {
		Error      ErrorCode
		Assignment []byte
	}

	HeartbeatRequest struct {
		GroupID      string
		GenerationID int32
		MemberID     string
	}

	HeartbeatResponse struct {
		Error ErrorCode
	}

	LeaveGroupRequest struct {
		GroupID  string
		MemberID string
	}

	LeaveGroupResponse struct {
		Error ErrorCode
	}

	ListGroupRequest struct {
	}

	ListGroupResponse struct {
		Error  ErrorCode
		Groups map[string]string
	}

	DescribeGroupsRequest struct {
		Groups []string
	}

	DescribeGroupsResponse struct {
		Groups map[string]GroupMetadata
	}

	GroupMetadata struct {
		Error        ErrorCode
		State        string
		ProtocolType string
		Protocol     string
		Members      []MemberInfo
	}

	MemberInfo struct {
		MemberID   string
		ClientID   string
		ClientHost string
		Metadata   []byte
		Assignment []byte
	}

	RawMessageSet struct {
		Payload []byte
	}
)

type ID int32

type ErrorCode uint16
