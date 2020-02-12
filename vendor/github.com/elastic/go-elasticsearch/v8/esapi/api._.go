// Code generated from specification version 8.0.0 (290fb706150): DO NOT EDIT

package esapi

// API contains the Elasticsearch APIs
//
type API struct {
	Cat        *Cat
	Cluster    *Cluster
	Indices    *Indices
	Ingest     *Ingest
	Nodes      *Nodes
	Remote     *Remote
	Snapshot   *Snapshot
	Tasks      *Tasks
	CCR        *CCR
	ILM        *ILM
	License    *License
	Migration  *Migration
	ML         *ML
	Monitoring *Monitoring
	Rollup     *Rollup
	Security   *Security
	SQL        *SQL
	SSL        *SSL
	Watcher    *Watcher
	XPack      *XPack

	Bulk                                          Bulk
	ClearScroll                                   ClearScroll
	Count                                         Count
	Create                                        Create
	DataFrameDeleteDataFrameTransform             DataFrameDeleteDataFrameTransform
	DataFrameGetDataFrameTransform                DataFrameGetDataFrameTransform
	DataFrameGetDataFrameTransformStats           DataFrameGetDataFrameTransformStats
	DataFramePreviewDataFrameTransform            DataFramePreviewDataFrameTransform
	DataFramePutDataFrameTransform                DataFramePutDataFrameTransform
	DataFrameStartDataFrameTransform              DataFrameStartDataFrameTransform
	DataFrameStopDataFrameTransform               DataFrameStopDataFrameTransform
	DataFrameTransformDeprecatedDeleteTransform   DataFrameTransformDeprecatedDeleteTransform
	DataFrameTransformDeprecatedGetTransform      DataFrameTransformDeprecatedGetTransform
	DataFrameTransformDeprecatedGetTransformStats DataFrameTransformDeprecatedGetTransformStats
	DataFrameTransformDeprecatedPreviewTransform  DataFrameTransformDeprecatedPreviewTransform
	DataFrameTransformDeprecatedPutTransform      DataFrameTransformDeprecatedPutTransform
	DataFrameTransformDeprecatedStartTransform    DataFrameTransformDeprecatedStartTransform
	DataFrameTransformDeprecatedStopTransform     DataFrameTransformDeprecatedStopTransform
	DataFrameTransformDeprecatedUpdateTransform   DataFrameTransformDeprecatedUpdateTransform
	DataFrameUpdateDataFrameTransform             DataFrameUpdateDataFrameTransform
	DeleteByQuery                                 DeleteByQuery
	DeleteByQueryRethrottle                       DeleteByQueryRethrottle
	Delete                                        Delete
	DeleteScript                                  DeleteScript
	EnrichDeletePolicy                            EnrichDeletePolicy
	EnrichExecutePolicy                           EnrichExecutePolicy
	EnrichGetPolicy                               EnrichGetPolicy
	EnrichPutPolicy                               EnrichPutPolicy
	EnrichStats                                   EnrichStats
	EqlSearch                                     EqlSearch
	Exists                                        Exists
	ExistsSource                                  ExistsSource
	Explain                                       Explain
	FieldCaps                                     FieldCaps
	Get                                           Get
	GetScriptContext                              GetScriptContext
	GetScriptLanguages                            GetScriptLanguages
	GetScript                                     GetScript
	GetSource                                     GetSource
	GraphExplore                                  GraphExplore
	Index                                         Index
	Info                                          Info
	Mget                                          Mget
	Msearch                                       Msearch
	MsearchTemplate                               MsearchTemplate
	Mtermvectors                                  Mtermvectors
	Ping                                          Ping
	PutScript                                     PutScript
	RankEval                                      RankEval
	Reindex                                       Reindex
	ReindexRethrottle                             ReindexRethrottle
	RenderSearchTemplate                          RenderSearchTemplate
	ScriptsPainlessContext                        ScriptsPainlessContext
	ScriptsPainlessExecute                        ScriptsPainlessExecute
	Scroll                                        Scroll
	Search                                        Search
	SearchShards                                  SearchShards
	SearchTemplate                                SearchTemplate
	SlmDeleteLifecycle                            SlmDeleteLifecycle
	SlmExecuteLifecycle                           SlmExecuteLifecycle
	SlmExecuteRetention                           SlmExecuteRetention
	SlmGetLifecycle                               SlmGetLifecycle
	SlmGetStats                                   SlmGetStats
	SlmGetStatus                                  SlmGetStatus
	SlmPutLifecycle                               SlmPutLifecycle
	SlmStart                                      SlmStart
	SlmStop                                       SlmStop
	Termvectors                                   Termvectors
	TransformDeleteTransform                      TransformDeleteTransform
	TransformGetTransform                         TransformGetTransform
	TransformGetTransformStats                    TransformGetTransformStats
	TransformPreviewTransform                     TransformPreviewTransform
	TransformPutTransform                         TransformPutTransform
	TransformStartTransform                       TransformStartTransform
	TransformStopTransform                        TransformStopTransform
	TransformUpdateTransform                      TransformUpdateTransform
	UpdateByQuery                                 UpdateByQuery
	UpdateByQueryRethrottle                       UpdateByQueryRethrottle
	Update                                        Update
}

// Cat contains the Cat APIs
type Cat struct {
	Aliases         CatAliases
	Allocation      CatAllocation
	Count           CatCount
	Fielddata       CatFielddata
	Health          CatHealth
	Help            CatHelp
	Indices         CatIndices
	MLDatafeeds     CatMLDatafeeds
	MLJobs          CatMLJobs
	MLTrainedModels CatMLTrainedModels
	Master          CatMaster
	Nodeattrs       CatNodeattrs
	Nodes           CatNodes
	PendingTasks    CatPendingTasks
	Plugins         CatPlugins
	Recovery        CatRecovery
	Repositories    CatRepositories
	Segments        CatSegments
	Shards          CatShards
	Snapshots       CatSnapshots
	Tasks           CatTasks
	Templates       CatTemplates
	ThreadPool      CatThreadPool
}

// Cluster contains the Cluster APIs
type Cluster struct {
	AllocationExplain ClusterAllocationExplain
	GetSettings       ClusterGetSettings
	Health            ClusterHealth
	PendingTasks      ClusterPendingTasks
	PutSettings       ClusterPutSettings
	RemoteInfo        ClusterRemoteInfo
	Reroute           ClusterReroute
	State             ClusterState
	Stats             ClusterStats
}

// Indices contains the Indices APIs
type Indices struct {
	Analyze               IndicesAnalyze
	ClearCache            IndicesClearCache
	Clone                 IndicesClone
	Close                 IndicesClose
	Create                IndicesCreate
	DeleteAlias           IndicesDeleteAlias
	Delete                IndicesDelete
	DeleteTemplate        IndicesDeleteTemplate
	ExistsAlias           IndicesExistsAlias
	ExistsDocumentType    IndicesExistsDocumentType
	Exists                IndicesExists
	ExistsTemplate        IndicesExistsTemplate
	Flush                 IndicesFlush
	FlushSynced           IndicesFlushSynced
	Forcemerge            IndicesForcemerge
	Freeze                IndicesFreeze
	GetAlias              IndicesGetAlias
	GetFieldMapping       IndicesGetFieldMapping
	GetMapping            IndicesGetMapping
	Get                   IndicesGet
	GetSettings           IndicesGetSettings
	GetTemplate           IndicesGetTemplate
	GetUpgrade            IndicesGetUpgrade
	Open                  IndicesOpen
	PutAlias              IndicesPutAlias
	PutMapping            IndicesPutMapping
	PutSettings           IndicesPutSettings
	PutTemplate           IndicesPutTemplate
	Recovery              IndicesRecovery
	Refresh               IndicesRefresh
	ReloadSearchAnalyzers IndicesReloadSearchAnalyzers
	Rollover              IndicesRollover
	Segments              IndicesSegments
	ShardStores           IndicesShardStores
	Shrink                IndicesShrink
	Split                 IndicesSplit
	Stats                 IndicesStats
	Unfreeze              IndicesUnfreeze
	UpdateAliases         IndicesUpdateAliases
	Upgrade               IndicesUpgrade
	ValidateQuery         IndicesValidateQuery
}

// Ingest contains the Ingest APIs
type Ingest struct {
	DeletePipeline IngestDeletePipeline
	GetPipeline    IngestGetPipeline
	ProcessorGrok  IngestProcessorGrok
	PutPipeline    IngestPutPipeline
	Simulate       IngestSimulate
}

// Nodes contains the Nodes APIs
type Nodes struct {
	HotThreads           NodesHotThreads
	Info                 NodesInfo
	ReloadSecureSettings NodesReloadSecureSettings
	Stats                NodesStats
	Usage                NodesUsage
}

// Remote contains the Remote APIs
type Remote struct {
}

// Snapshot contains the Snapshot APIs
type Snapshot struct {
	CleanupRepository SnapshotCleanupRepository
	CreateRepository  SnapshotCreateRepository
	Create            SnapshotCreate
	DeleteRepository  SnapshotDeleteRepository
	Delete            SnapshotDelete
	GetRepository     SnapshotGetRepository
	Get               SnapshotGet
	Restore           SnapshotRestore
	Status            SnapshotStatus
	VerifyRepository  SnapshotVerifyRepository
}

// Tasks contains the Tasks APIs
type Tasks struct {
	Cancel TasksCancel
	Get    TasksGet
	List   TasksList
}

// CCR contains the CCR APIs
type CCR struct {
	DeleteAutoFollowPattern CCRDeleteAutoFollowPattern
	FollowInfo              CCRFollowInfo
	Follow                  CCRFollow
	FollowStats             CCRFollowStats
	ForgetFollower          CCRForgetFollower
	GetAutoFollowPattern    CCRGetAutoFollowPattern
	PauseAutoFollowPattern  CCRPauseAutoFollowPattern
	PauseFollow             CCRPauseFollow
	PutAutoFollowPattern    CCRPutAutoFollowPattern
	ResumeAutoFollowPattern CCRResumeAutoFollowPattern
	ResumeFollow            CCRResumeFollow
	Stats                   CCRStats
	Unfollow                CCRUnfollow
}

// ILM contains the ILM APIs
type ILM struct {
	DeleteLifecycle  ILMDeleteLifecycle
	ExplainLifecycle ILMExplainLifecycle
	GetLifecycle     ILMGetLifecycle
	GetStatus        ILMGetStatus
	MoveToStep       ILMMoveToStep
	PutLifecycle     ILMPutLifecycle
	RemovePolicy     ILMRemovePolicy
	Retry            ILMRetry
	Start            ILMStart
	Stop             ILMStop
}

// License contains the License APIs
type License struct {
	Delete         LicenseDelete
	GetBasicStatus LicenseGetBasicStatus
	Get            LicenseGet
	GetTrialStatus LicenseGetTrialStatus
	Post           LicensePost
	PostStartBasic LicensePostStartBasic
	PostStartTrial LicensePostStartTrial
}

// Migration contains the Migration APIs
type Migration struct {
	Deprecations MigrationDeprecations
}

// ML contains the ML APIs
type ML struct {
	CloseJob                   MLCloseJob
	DeleteCalendarEvent        MLDeleteCalendarEvent
	DeleteCalendarJob          MLDeleteCalendarJob
	DeleteCalendar             MLDeleteCalendar
	DeleteDataFrameAnalytics   MLDeleteDataFrameAnalytics
	DeleteDatafeed             MLDeleteDatafeed
	DeleteExpiredData          MLDeleteExpiredData
	DeleteFilter               MLDeleteFilter
	DeleteForecast             MLDeleteForecast
	DeleteJob                  MLDeleteJob
	DeleteModelSnapshot        MLDeleteModelSnapshot
	DeleteTrainedModel         MLDeleteTrainedModel
	EstimateMemoryUsage        MLEstimateMemoryUsage
	EvaluateDataFrame          MLEvaluateDataFrame
	ExplainDataFrameAnalytics  MLExplainDataFrameAnalytics
	FindFileStructure          MLFindFileStructure
	FlushJob                   MLFlushJob
	Forecast                   MLForecast
	GetBuckets                 MLGetBuckets
	GetCalendarEvents          MLGetCalendarEvents
	GetCalendars               MLGetCalendars
	GetCategories              MLGetCategories
	GetDataFrameAnalytics      MLGetDataFrameAnalytics
	GetDataFrameAnalyticsStats MLGetDataFrameAnalyticsStats
	GetDatafeedStats           MLGetDatafeedStats
	GetDatafeeds               MLGetDatafeeds
	GetFilters                 MLGetFilters
	GetInfluencers             MLGetInfluencers
	GetJobStats                MLGetJobStats
	GetJobs                    MLGetJobs
	GetModelSnapshots          MLGetModelSnapshots
	GetOverallBuckets          MLGetOverallBuckets
	GetRecords                 MLGetRecords
	GetTrainedModels           MLGetTrainedModels
	GetTrainedModelsStats      MLGetTrainedModelsStats
	Info                       MLInfo
	OpenJob                    MLOpenJob
	PostCalendarEvents         MLPostCalendarEvents
	PostData                   MLPostData
	PreviewDatafeed            MLPreviewDatafeed
	PutCalendarJob             MLPutCalendarJob
	PutCalendar                MLPutCalendar
	PutDataFrameAnalytics      MLPutDataFrameAnalytics
	PutDatafeed                MLPutDatafeed
	PutFilter                  MLPutFilter
	PutJob                     MLPutJob
	PutTrainedModel            MLPutTrainedModel
	RevertModelSnapshot        MLRevertModelSnapshot
	SetUpgradeMode             MLSetUpgradeMode
	StartDataFrameAnalytics    MLStartDataFrameAnalytics
	StartDatafeed              MLStartDatafeed
	StopDataFrameAnalytics     MLStopDataFrameAnalytics
	StopDatafeed               MLStopDatafeed
	UpdateDatafeed             MLUpdateDatafeed
	UpdateFilter               MLUpdateFilter
	UpdateJob                  MLUpdateJob
	UpdateModelSnapshot        MLUpdateModelSnapshot
	ValidateDetector           MLValidateDetector
	Validate                   MLValidate
}

// Monitoring contains the Monitoring APIs
type Monitoring struct {
	Bulk MonitoringBulk
}

// Rollup contains the Rollup APIs
type Rollup struct {
	DeleteJob    RollupDeleteJob
	GetJobs      RollupGetJobs
	GetCaps      RollupGetRollupCaps
	GetIndexCaps RollupGetRollupIndexCaps
	PutJob       RollupPutJob
	Search       RollupRollupSearch
	StartJob     RollupStartJob
	StopJob      RollupStopJob
}

// Security contains the Security APIs
type Security struct {
	Authenticate         SecurityAuthenticate
	ChangePassword       SecurityChangePassword
	ClearCachedRealms    SecurityClearCachedRealms
	ClearCachedRoles     SecurityClearCachedRoles
	CreateAPIKey         SecurityCreateAPIKey
	DeletePrivileges     SecurityDeletePrivileges
	DeleteRoleMapping    SecurityDeleteRoleMapping
	DeleteRole           SecurityDeleteRole
	DeleteUser           SecurityDeleteUser
	DisableUser          SecurityDisableUser
	EnableUser           SecurityEnableUser
	GetAPIKey            SecurityGetAPIKey
	GetBuiltinPrivileges SecurityGetBuiltinPrivileges
	GetPrivileges        SecurityGetPrivileges
	GetRoleMapping       SecurityGetRoleMapping
	GetRole              SecurityGetRole
	GetToken             SecurityGetToken
	GetUserPrivileges    SecurityGetUserPrivileges
	GetUser              SecurityGetUser
	HasPrivileges        SecurityHasPrivileges
	InvalidateAPIKey     SecurityInvalidateAPIKey
	InvalidateToken      SecurityInvalidateToken
	PutPrivileges        SecurityPutPrivileges
	PutRoleMapping       SecurityPutRoleMapping
	PutRole              SecurityPutRole
	PutUser              SecurityPutUser
}

// SQL contains the SQL APIs
type SQL struct {
	ClearCursor SQLClearCursor
	Query       SQLQuery
	Translate   SQLTranslate
}

// SSL contains the SSL APIs
type SSL struct {
	Certificates SSLCertificates
}

// Watcher contains the Watcher APIs
type Watcher struct {
	AckWatch        WatcherAckWatch
	ActivateWatch   WatcherActivateWatch
	DeactivateWatch WatcherDeactivateWatch
	DeleteWatch     WatcherDeleteWatch
	ExecuteWatch    WatcherExecuteWatch
	GetWatch        WatcherGetWatch
	PutWatch        WatcherPutWatch
	Start           WatcherStart
	Stats           WatcherStats
	Stop            WatcherStop
}

// XPack contains the XPack APIs
type XPack struct {
	Info  XPackInfo
	Usage XPackUsage
}

// New creates new API
//
func New(t Transport) *API {
	return &API{
		Bulk:                                          newBulkFunc(t),
		ClearScroll:                                   newClearScrollFunc(t),
		Count:                                         newCountFunc(t),
		Create:                                        newCreateFunc(t),
		DataFrameDeleteDataFrameTransform:             newDataFrameDeleteDataFrameTransformFunc(t),
		DataFrameGetDataFrameTransform:                newDataFrameGetDataFrameTransformFunc(t),
		DataFrameGetDataFrameTransformStats:           newDataFrameGetDataFrameTransformStatsFunc(t),
		DataFramePreviewDataFrameTransform:            newDataFramePreviewDataFrameTransformFunc(t),
		DataFramePutDataFrameTransform:                newDataFramePutDataFrameTransformFunc(t),
		DataFrameStartDataFrameTransform:              newDataFrameStartDataFrameTransformFunc(t),
		DataFrameStopDataFrameTransform:               newDataFrameStopDataFrameTransformFunc(t),
		DataFrameTransformDeprecatedDeleteTransform:   newDataFrameTransformDeprecatedDeleteTransformFunc(t),
		DataFrameTransformDeprecatedGetTransform:      newDataFrameTransformDeprecatedGetTransformFunc(t),
		DataFrameTransformDeprecatedGetTransformStats: newDataFrameTransformDeprecatedGetTransformStatsFunc(t),
		DataFrameTransformDeprecatedPreviewTransform:  newDataFrameTransformDeprecatedPreviewTransformFunc(t),
		DataFrameTransformDeprecatedPutTransform:      newDataFrameTransformDeprecatedPutTransformFunc(t),
		DataFrameTransformDeprecatedStartTransform:    newDataFrameTransformDeprecatedStartTransformFunc(t),
		DataFrameTransformDeprecatedStopTransform:     newDataFrameTransformDeprecatedStopTransformFunc(t),
		DataFrameTransformDeprecatedUpdateTransform:   newDataFrameTransformDeprecatedUpdateTransformFunc(t),
		DataFrameUpdateDataFrameTransform:             newDataFrameUpdateDataFrameTransformFunc(t),
		DeleteByQuery:                                 newDeleteByQueryFunc(t),
		DeleteByQueryRethrottle:                       newDeleteByQueryRethrottleFunc(t),
		Delete:                                        newDeleteFunc(t),
		DeleteScript:                                  newDeleteScriptFunc(t),
		EnrichDeletePolicy:                            newEnrichDeletePolicyFunc(t),
		EnrichExecutePolicy:                           newEnrichExecutePolicyFunc(t),
		EnrichGetPolicy:                               newEnrichGetPolicyFunc(t),
		EnrichPutPolicy:                               newEnrichPutPolicyFunc(t),
		EnrichStats:                                   newEnrichStatsFunc(t),
		EqlSearch:                                     newEqlSearchFunc(t),
		Exists:                                        newExistsFunc(t),
		ExistsSource:                                  newExistsSourceFunc(t),
		Explain:                                       newExplainFunc(t),
		FieldCaps:                                     newFieldCapsFunc(t),
		Get:                                           newGetFunc(t),
		GetScriptContext:                              newGetScriptContextFunc(t),
		GetScriptLanguages:                            newGetScriptLanguagesFunc(t),
		GetScript:                                     newGetScriptFunc(t),
		GetSource:                                     newGetSourceFunc(t),
		GraphExplore:                                  newGraphExploreFunc(t),
		Index:                                         newIndexFunc(t),
		Info:                                          newInfoFunc(t),
		Mget:                                          newMgetFunc(t),
		Msearch:                                       newMsearchFunc(t),
		MsearchTemplate:                               newMsearchTemplateFunc(t),
		Mtermvectors:                                  newMtermvectorsFunc(t),
		Ping:                                          newPingFunc(t),
		PutScript:                                     newPutScriptFunc(t),
		RankEval:                                      newRankEvalFunc(t),
		Reindex:                                       newReindexFunc(t),
		ReindexRethrottle:                             newReindexRethrottleFunc(t),
		RenderSearchTemplate:                          newRenderSearchTemplateFunc(t),
		ScriptsPainlessContext:                        newScriptsPainlessContextFunc(t),
		ScriptsPainlessExecute:                        newScriptsPainlessExecuteFunc(t),
		Scroll:                                        newScrollFunc(t),
		Search:                                        newSearchFunc(t),
		SearchShards:                                  newSearchShardsFunc(t),
		SearchTemplate:                                newSearchTemplateFunc(t),
		SlmDeleteLifecycle:                            newSlmDeleteLifecycleFunc(t),
		SlmExecuteLifecycle:                           newSlmExecuteLifecycleFunc(t),
		SlmExecuteRetention:                           newSlmExecuteRetentionFunc(t),
		SlmGetLifecycle:                               newSlmGetLifecycleFunc(t),
		SlmGetStats:                                   newSlmGetStatsFunc(t),
		SlmGetStatus:                                  newSlmGetStatusFunc(t),
		SlmPutLifecycle:                               newSlmPutLifecycleFunc(t),
		SlmStart:                                      newSlmStartFunc(t),
		SlmStop:                                       newSlmStopFunc(t),
		Termvectors:                                   newTermvectorsFunc(t),
		TransformDeleteTransform:                      newTransformDeleteTransformFunc(t),
		TransformGetTransform:                         newTransformGetTransformFunc(t),
		TransformGetTransformStats:                    newTransformGetTransformStatsFunc(t),
		TransformPreviewTransform:                     newTransformPreviewTransformFunc(t),
		TransformPutTransform:                         newTransformPutTransformFunc(t),
		TransformStartTransform:                       newTransformStartTransformFunc(t),
		TransformStopTransform:                        newTransformStopTransformFunc(t),
		TransformUpdateTransform:                      newTransformUpdateTransformFunc(t),
		UpdateByQuery:                                 newUpdateByQueryFunc(t),
		UpdateByQueryRethrottle:                       newUpdateByQueryRethrottleFunc(t),
		Update:                                        newUpdateFunc(t),
		Cat: &Cat{
			Aliases:         newCatAliasesFunc(t),
			Allocation:      newCatAllocationFunc(t),
			Count:           newCatCountFunc(t),
			Fielddata:       newCatFielddataFunc(t),
			Health:          newCatHealthFunc(t),
			Help:            newCatHelpFunc(t),
			Indices:         newCatIndicesFunc(t),
			MLDatafeeds:     newCatMLDatafeedsFunc(t),
			MLJobs:          newCatMLJobsFunc(t),
			MLTrainedModels: newCatMLTrainedModelsFunc(t),
			Master:          newCatMasterFunc(t),
			Nodeattrs:       newCatNodeattrsFunc(t),
			Nodes:           newCatNodesFunc(t),
			PendingTasks:    newCatPendingTasksFunc(t),
			Plugins:         newCatPluginsFunc(t),
			Recovery:        newCatRecoveryFunc(t),
			Repositories:    newCatRepositoriesFunc(t),
			Segments:        newCatSegmentsFunc(t),
			Shards:          newCatShardsFunc(t),
			Snapshots:       newCatSnapshotsFunc(t),
			Tasks:           newCatTasksFunc(t),
			Templates:       newCatTemplatesFunc(t),
			ThreadPool:      newCatThreadPoolFunc(t),
		},
		Cluster: &Cluster{
			AllocationExplain: newClusterAllocationExplainFunc(t),
			GetSettings:       newClusterGetSettingsFunc(t),
			Health:            newClusterHealthFunc(t),
			PendingTasks:      newClusterPendingTasksFunc(t),
			PutSettings:       newClusterPutSettingsFunc(t),
			RemoteInfo:        newClusterRemoteInfoFunc(t),
			Reroute:           newClusterRerouteFunc(t),
			State:             newClusterStateFunc(t),
			Stats:             newClusterStatsFunc(t),
		},
		Indices: &Indices{
			Analyze:               newIndicesAnalyzeFunc(t),
			ClearCache:            newIndicesClearCacheFunc(t),
			Clone:                 newIndicesCloneFunc(t),
			Close:                 newIndicesCloseFunc(t),
			Create:                newIndicesCreateFunc(t),
			DeleteAlias:           newIndicesDeleteAliasFunc(t),
			Delete:                newIndicesDeleteFunc(t),
			DeleteTemplate:        newIndicesDeleteTemplateFunc(t),
			ExistsAlias:           newIndicesExistsAliasFunc(t),
			ExistsDocumentType:    newIndicesExistsDocumentTypeFunc(t),
			Exists:                newIndicesExistsFunc(t),
			ExistsTemplate:        newIndicesExistsTemplateFunc(t),
			Flush:                 newIndicesFlushFunc(t),
			FlushSynced:           newIndicesFlushSyncedFunc(t),
			Forcemerge:            newIndicesForcemergeFunc(t),
			Freeze:                newIndicesFreezeFunc(t),
			GetAlias:              newIndicesGetAliasFunc(t),
			GetFieldMapping:       newIndicesGetFieldMappingFunc(t),
			GetMapping:            newIndicesGetMappingFunc(t),
			Get:                   newIndicesGetFunc(t),
			GetSettings:           newIndicesGetSettingsFunc(t),
			GetTemplate:           newIndicesGetTemplateFunc(t),
			GetUpgrade:            newIndicesGetUpgradeFunc(t),
			Open:                  newIndicesOpenFunc(t),
			PutAlias:              newIndicesPutAliasFunc(t),
			PutMapping:            newIndicesPutMappingFunc(t),
			PutSettings:           newIndicesPutSettingsFunc(t),
			PutTemplate:           newIndicesPutTemplateFunc(t),
			Recovery:              newIndicesRecoveryFunc(t),
			Refresh:               newIndicesRefreshFunc(t),
			ReloadSearchAnalyzers: newIndicesReloadSearchAnalyzersFunc(t),
			Rollover:              newIndicesRolloverFunc(t),
			Segments:              newIndicesSegmentsFunc(t),
			ShardStores:           newIndicesShardStoresFunc(t),
			Shrink:                newIndicesShrinkFunc(t),
			Split:                 newIndicesSplitFunc(t),
			Stats:                 newIndicesStatsFunc(t),
			Unfreeze:              newIndicesUnfreezeFunc(t),
			UpdateAliases:         newIndicesUpdateAliasesFunc(t),
			Upgrade:               newIndicesUpgradeFunc(t),
			ValidateQuery:         newIndicesValidateQueryFunc(t),
		},
		Ingest: &Ingest{
			DeletePipeline: newIngestDeletePipelineFunc(t),
			GetPipeline:    newIngestGetPipelineFunc(t),
			ProcessorGrok:  newIngestProcessorGrokFunc(t),
			PutPipeline:    newIngestPutPipelineFunc(t),
			Simulate:       newIngestSimulateFunc(t),
		},
		Nodes: &Nodes{
			HotThreads:           newNodesHotThreadsFunc(t),
			Info:                 newNodesInfoFunc(t),
			ReloadSecureSettings: newNodesReloadSecureSettingsFunc(t),
			Stats:                newNodesStatsFunc(t),
			Usage:                newNodesUsageFunc(t),
		},
		Remote: &Remote{},
		Snapshot: &Snapshot{
			CleanupRepository: newSnapshotCleanupRepositoryFunc(t),
			CreateRepository:  newSnapshotCreateRepositoryFunc(t),
			Create:            newSnapshotCreateFunc(t),
			DeleteRepository:  newSnapshotDeleteRepositoryFunc(t),
			Delete:            newSnapshotDeleteFunc(t),
			GetRepository:     newSnapshotGetRepositoryFunc(t),
			Get:               newSnapshotGetFunc(t),
			Restore:           newSnapshotRestoreFunc(t),
			Status:            newSnapshotStatusFunc(t),
			VerifyRepository:  newSnapshotVerifyRepositoryFunc(t),
		},
		Tasks: &Tasks{
			Cancel: newTasksCancelFunc(t),
			Get:    newTasksGetFunc(t),
			List:   newTasksListFunc(t),
		},
		CCR: &CCR{
			DeleteAutoFollowPattern: newCCRDeleteAutoFollowPatternFunc(t),
			FollowInfo:              newCCRFollowInfoFunc(t),
			Follow:                  newCCRFollowFunc(t),
			FollowStats:             newCCRFollowStatsFunc(t),
			ForgetFollower:          newCCRForgetFollowerFunc(t),
			GetAutoFollowPattern:    newCCRGetAutoFollowPatternFunc(t),
			PauseAutoFollowPattern:  newCCRPauseAutoFollowPatternFunc(t),
			PauseFollow:             newCCRPauseFollowFunc(t),
			PutAutoFollowPattern:    newCCRPutAutoFollowPatternFunc(t),
			ResumeAutoFollowPattern: newCCRResumeAutoFollowPatternFunc(t),
			ResumeFollow:            newCCRResumeFollowFunc(t),
			Stats:                   newCCRStatsFunc(t),
			Unfollow:                newCCRUnfollowFunc(t),
		},
		ILM: &ILM{
			DeleteLifecycle:  newILMDeleteLifecycleFunc(t),
			ExplainLifecycle: newILMExplainLifecycleFunc(t),
			GetLifecycle:     newILMGetLifecycleFunc(t),
			GetStatus:        newILMGetStatusFunc(t),
			MoveToStep:       newILMMoveToStepFunc(t),
			PutLifecycle:     newILMPutLifecycleFunc(t),
			RemovePolicy:     newILMRemovePolicyFunc(t),
			Retry:            newILMRetryFunc(t),
			Start:            newILMStartFunc(t),
			Stop:             newILMStopFunc(t),
		},
		License: &License{
			Delete:         newLicenseDeleteFunc(t),
			GetBasicStatus: newLicenseGetBasicStatusFunc(t),
			Get:            newLicenseGetFunc(t),
			GetTrialStatus: newLicenseGetTrialStatusFunc(t),
			Post:           newLicensePostFunc(t),
			PostStartBasic: newLicensePostStartBasicFunc(t),
			PostStartTrial: newLicensePostStartTrialFunc(t),
		},
		Migration: &Migration{
			Deprecations: newMigrationDeprecationsFunc(t),
		},
		ML: &ML{
			CloseJob:                   newMLCloseJobFunc(t),
			DeleteCalendarEvent:        newMLDeleteCalendarEventFunc(t),
			DeleteCalendarJob:          newMLDeleteCalendarJobFunc(t),
			DeleteCalendar:             newMLDeleteCalendarFunc(t),
			DeleteDataFrameAnalytics:   newMLDeleteDataFrameAnalyticsFunc(t),
			DeleteDatafeed:             newMLDeleteDatafeedFunc(t),
			DeleteExpiredData:          newMLDeleteExpiredDataFunc(t),
			DeleteFilter:               newMLDeleteFilterFunc(t),
			DeleteForecast:             newMLDeleteForecastFunc(t),
			DeleteJob:                  newMLDeleteJobFunc(t),
			DeleteModelSnapshot:        newMLDeleteModelSnapshotFunc(t),
			DeleteTrainedModel:         newMLDeleteTrainedModelFunc(t),
			EstimateMemoryUsage:        newMLEstimateMemoryUsageFunc(t),
			EvaluateDataFrame:          newMLEvaluateDataFrameFunc(t),
			ExplainDataFrameAnalytics:  newMLExplainDataFrameAnalyticsFunc(t),
			FindFileStructure:          newMLFindFileStructureFunc(t),
			FlushJob:                   newMLFlushJobFunc(t),
			Forecast:                   newMLForecastFunc(t),
			GetBuckets:                 newMLGetBucketsFunc(t),
			GetCalendarEvents:          newMLGetCalendarEventsFunc(t),
			GetCalendars:               newMLGetCalendarsFunc(t),
			GetCategories:              newMLGetCategoriesFunc(t),
			GetDataFrameAnalytics:      newMLGetDataFrameAnalyticsFunc(t),
			GetDataFrameAnalyticsStats: newMLGetDataFrameAnalyticsStatsFunc(t),
			GetDatafeedStats:           newMLGetDatafeedStatsFunc(t),
			GetDatafeeds:               newMLGetDatafeedsFunc(t),
			GetFilters:                 newMLGetFiltersFunc(t),
			GetInfluencers:             newMLGetInfluencersFunc(t),
			GetJobStats:                newMLGetJobStatsFunc(t),
			GetJobs:                    newMLGetJobsFunc(t),
			GetModelSnapshots:          newMLGetModelSnapshotsFunc(t),
			GetOverallBuckets:          newMLGetOverallBucketsFunc(t),
			GetRecords:                 newMLGetRecordsFunc(t),
			GetTrainedModels:           newMLGetTrainedModelsFunc(t),
			GetTrainedModelsStats:      newMLGetTrainedModelsStatsFunc(t),
			Info:                       newMLInfoFunc(t),
			OpenJob:                    newMLOpenJobFunc(t),
			PostCalendarEvents:         newMLPostCalendarEventsFunc(t),
			PostData:                   newMLPostDataFunc(t),
			PreviewDatafeed:            newMLPreviewDatafeedFunc(t),
			PutCalendarJob:             newMLPutCalendarJobFunc(t),
			PutCalendar:                newMLPutCalendarFunc(t),
			PutDataFrameAnalytics:      newMLPutDataFrameAnalyticsFunc(t),
			PutDatafeed:                newMLPutDatafeedFunc(t),
			PutFilter:                  newMLPutFilterFunc(t),
			PutJob:                     newMLPutJobFunc(t),
			PutTrainedModel:            newMLPutTrainedModelFunc(t),
			RevertModelSnapshot:        newMLRevertModelSnapshotFunc(t),
			SetUpgradeMode:             newMLSetUpgradeModeFunc(t),
			StartDataFrameAnalytics:    newMLStartDataFrameAnalyticsFunc(t),
			StartDatafeed:              newMLStartDatafeedFunc(t),
			StopDataFrameAnalytics:     newMLStopDataFrameAnalyticsFunc(t),
			StopDatafeed:               newMLStopDatafeedFunc(t),
			UpdateDatafeed:             newMLUpdateDatafeedFunc(t),
			UpdateFilter:               newMLUpdateFilterFunc(t),
			UpdateJob:                  newMLUpdateJobFunc(t),
			UpdateModelSnapshot:        newMLUpdateModelSnapshotFunc(t),
			ValidateDetector:           newMLValidateDetectorFunc(t),
			Validate:                   newMLValidateFunc(t),
		},
		Monitoring: &Monitoring{
			Bulk: newMonitoringBulkFunc(t),
		},
		Rollup: &Rollup{
			DeleteJob:    newRollupDeleteJobFunc(t),
			GetJobs:      newRollupGetJobsFunc(t),
			GetCaps:      newRollupGetRollupCapsFunc(t),
			GetIndexCaps: newRollupGetRollupIndexCapsFunc(t),
			PutJob:       newRollupPutJobFunc(t),
			Search:       newRollupRollupSearchFunc(t),
			StartJob:     newRollupStartJobFunc(t),
			StopJob:      newRollupStopJobFunc(t),
		},
		Security: &Security{
			Authenticate:         newSecurityAuthenticateFunc(t),
			ChangePassword:       newSecurityChangePasswordFunc(t),
			ClearCachedRealms:    newSecurityClearCachedRealmsFunc(t),
			ClearCachedRoles:     newSecurityClearCachedRolesFunc(t),
			CreateAPIKey:         newSecurityCreateAPIKeyFunc(t),
			DeletePrivileges:     newSecurityDeletePrivilegesFunc(t),
			DeleteRoleMapping:    newSecurityDeleteRoleMappingFunc(t),
			DeleteRole:           newSecurityDeleteRoleFunc(t),
			DeleteUser:           newSecurityDeleteUserFunc(t),
			DisableUser:          newSecurityDisableUserFunc(t),
			EnableUser:           newSecurityEnableUserFunc(t),
			GetAPIKey:            newSecurityGetAPIKeyFunc(t),
			GetBuiltinPrivileges: newSecurityGetBuiltinPrivilegesFunc(t),
			GetPrivileges:        newSecurityGetPrivilegesFunc(t),
			GetRoleMapping:       newSecurityGetRoleMappingFunc(t),
			GetRole:              newSecurityGetRoleFunc(t),
			GetToken:             newSecurityGetTokenFunc(t),
			GetUserPrivileges:    newSecurityGetUserPrivilegesFunc(t),
			GetUser:              newSecurityGetUserFunc(t),
			HasPrivileges:        newSecurityHasPrivilegesFunc(t),
			InvalidateAPIKey:     newSecurityInvalidateAPIKeyFunc(t),
			InvalidateToken:      newSecurityInvalidateTokenFunc(t),
			PutPrivileges:        newSecurityPutPrivilegesFunc(t),
			PutRoleMapping:       newSecurityPutRoleMappingFunc(t),
			PutRole:              newSecurityPutRoleFunc(t),
			PutUser:              newSecurityPutUserFunc(t),
		},
		SQL: &SQL{
			ClearCursor: newSQLClearCursorFunc(t),
			Query:       newSQLQueryFunc(t),
			Translate:   newSQLTranslateFunc(t),
		},
		SSL: &SSL{
			Certificates: newSSLCertificatesFunc(t),
		},
		Watcher: &Watcher{
			AckWatch:        newWatcherAckWatchFunc(t),
			ActivateWatch:   newWatcherActivateWatchFunc(t),
			DeactivateWatch: newWatcherDeactivateWatchFunc(t),
			DeleteWatch:     newWatcherDeleteWatchFunc(t),
			ExecuteWatch:    newWatcherExecuteWatchFunc(t),
			GetWatch:        newWatcherGetWatchFunc(t),
			PutWatch:        newWatcherPutWatchFunc(t),
			Start:           newWatcherStartFunc(t),
			Stats:           newWatcherStatsFunc(t),
			Stop:            newWatcherStopFunc(t),
		},
		XPack: &XPack{
			Info:  newXPackInfoFunc(t),
			Usage: newXPackUsageFunc(t),
		},
	}
}
