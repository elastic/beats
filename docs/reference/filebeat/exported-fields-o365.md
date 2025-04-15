---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-o365.html
---

# Office 365 fields [exported-fields-o365]

Module for handling logs from Office 365.


## o365.audit [_o365_audit]

Fields from Office 365 Management API audit logs.

**`o365.audit.AADGroupId`**
:   type: keyword


**`o365.audit.Activity`**
:   type: keyword


**`o365.audit.Actor`**
:   type: array


**`o365.audit.ActorContextId`**
:   type: keyword


**`o365.audit.ActorIpAddress`**
:   type: keyword


**`o365.audit.ActorUserId`**
:   type: keyword


**`o365.audit.ActorYammerUserId`**
:   type: keyword


**`o365.audit.AlertEntityId`**
:   type: keyword


**`o365.audit.AlertId`**
:   type: keyword


**`o365.audit.AlertLinks`**
:   type: array


**`o365.audit.AlertType`**
:   type: keyword


**`o365.audit.AppId`**
:   type: keyword


**`o365.audit.ApplicationDisplayName`**
:   type: keyword


**`o365.audit.ApplicationId`**
:   type: keyword


**`o365.audit.AzureActiveDirectoryEventType`**
:   type: keyword


**`o365.audit.ExchangeMetaData.*`**
:   type: object


**`o365.audit.Category`**
:   type: keyword


**`o365.audit.ClientAppId`**
:   type: keyword


**`o365.audit.ClientInfoString`**
:   type: keyword


**`o365.audit.ClientIP`**
:   type: keyword


**`o365.audit.ClientIPAddress`**
:   type: keyword


**`o365.audit.Comments`**
:   type: text


**`o365.audit.CommunicationType`**
:   type: keyword


**`o365.audit.CorrelationId`**
:   type: keyword


**`o365.audit.CreationTime`**
:   type: keyword


**`o365.audit.CustomUniqueId`**
:   type: keyword


**`o365.audit.Data`**
:   type: keyword


**`o365.audit.DataType`**
:   type: keyword


**`o365.audit.DoNotDistributeEvent`**
:   type: boolean


**`o365.audit.EntityType`**
:   type: keyword


**`o365.audit.ErrorNumber`**
:   type: keyword


**`o365.audit.EventData`**
:   type: keyword


**`o365.audit.EventSource`**
:   type: keyword


**`o365.audit.ExceptionInfo.*`**
:   type: object


**`o365.audit.Experience`**
:   type: keyword


**`o365.audit.ExtendedProperties.*`**
:   type: object


**`o365.audit.ExternalAccess`**
:   type: keyword


**`o365.audit.FromApp`**
:   type: boolean


**`o365.audit.GroupName`**
:   type: keyword


**`o365.audit.Id`**
:   type: keyword


**`o365.audit.ImplicitShare`**
:   type: keyword


**`o365.audit.IncidentId`**
:   type: keyword


**`o365.audit.InternalLogonType`**
:   type: keyword


**`o365.audit.InterSystemsId`**
:   type: keyword


**`o365.audit.IntraSystemId`**
:   type: keyword


**`o365.audit.IsDocLib`**
:   type: boolean


**`o365.audit.Item.*`**
:   type: object


**`o365.audit.Item.*.*`**
:   type: object


**`o365.audit.ItemCount`**
:   type: long


**`o365.audit.ItemName`**
:   type: keyword


**`o365.audit.ItemType`**
:   type: keyword


**`o365.audit.ListBaseTemplateType`**
:   type: keyword


**`o365.audit.ListBaseType`**
:   type: keyword


**`o365.audit.ListColor`**
:   type: keyword


**`o365.audit.ListIcon`**
:   type: keyword


**`o365.audit.ListId`**
:   type: keyword


**`o365.audit.ListTitle`**
:   type: keyword


**`o365.audit.ListItemUniqueId`**
:   type: keyword


**`o365.audit.LogonError`**
:   type: keyword


**`o365.audit.LogonType`**
:   type: keyword


**`o365.audit.LogonUserSid`**
:   type: keyword


**`o365.audit.MailboxGuid`**
:   type: keyword


**`o365.audit.MailboxOwnerMasterAccountSid`**
:   type: keyword


**`o365.audit.MailboxOwnerSid`**
:   type: keyword


**`o365.audit.MailboxOwnerUPN`**
:   type: keyword


**`o365.audit.Members`**
:   type: array


**`o365.audit.Members.*`**
:   type: object


**`o365.audit.ModifiedProperties.*.*`**
:   type: object


**`o365.audit.Name`**
:   type: keyword


**`o365.audit.ObjectId`**
:   type: keyword


**`o365.audit.ObjectDisplayName`**
:   type: keyword


**`o365.audit.ObjectType`**
:   type: keyword


**`o365.audit.Operation`**
:   type: keyword


**`o365.audit.OperationId`**
:   type: keyword


**`o365.audit.OperationProperties`**
:   type: object


**`o365.audit.OrganizationId`**
:   type: keyword


**`o365.audit.OrganizationName`**
:   type: keyword


**`o365.audit.OriginatingServer`**
:   type: keyword


**`o365.audit.Parameters.*`**
:   type: object


**`o365.audit.PolicyDetails`**
:   type: array


**`o365.audit.PolicyId`**
:   type: keyword


**`o365.audit.RecordType`**
:   type: keyword


**`o365.audit.RequestId`**
:   type: keyword


**`o365.audit.ResultStatus`**
:   type: keyword


**`o365.audit.SensitiveInfoDetectionIsIncluded`**
:   type: keyword


**`o365.audit.SharePointMetaData.*`**
:   type: object


**`o365.audit.SessionId`**
:   type: keyword


**`o365.audit.Severity`**
:   type: keyword


**`o365.audit.Site`**
:   type: keyword


**`o365.audit.SiteUrl`**
:   type: keyword


**`o365.audit.Source`**
:   type: keyword


**`o365.audit.SourceFileExtension`**
:   type: keyword


**`o365.audit.SourceFileName`**
:   type: keyword


**`o365.audit.SourceRelativeUrl`**
:   type: keyword


**`o365.audit.Status`**
:   type: keyword


**`o365.audit.SupportTicketId`**
:   type: keyword


**`o365.audit.Target`**
:   type: array


**`o365.audit.TargetContextId`**
:   type: keyword


**`o365.audit.TargetUserOrGroupName`**
:   type: keyword


**`o365.audit.TargetUserOrGroupType`**
:   type: keyword


**`o365.audit.TeamName`**
:   type: keyword


**`o365.audit.TeamGuid`**
:   type: keyword


**`o365.audit.TemplateTypeId`**
:   type: keyword


**`o365.audit.Timestamp`**
:   type: keyword


**`o365.audit.UniqueSharingId`**
:   type: keyword


**`o365.audit.UserAgent`**
:   type: keyword


**`o365.audit.UserId`**
:   type: keyword


**`o365.audit.UserKey`**
:   type: keyword


**`o365.audit.UserType`**
:   type: keyword


**`o365.audit.Version`**
:   type: keyword


**`o365.audit.WebId`**
:   type: keyword


**`o365.audit.Workload`**
:   type: keyword


**`o365.audit.WorkspaceId`**
:   type: keyword


**`o365.audit.WorkspaceName`**
:   type: keyword


**`o365.audit.YammerNetworkId`**
:   type: keyword


