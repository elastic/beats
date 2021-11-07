package cfclient

import (
	"io"
	"net/http"
	"net/url"
)

// CloudFoundryClient provides a baseline reference for the currently supported APIs with Cloud Foundry. It can be used
// for providing custom implementations or extensions beyond the base implementation provided with this package.
type CloudFoundryClient interface {
	ListAllProcesses() ([]Process, error)
	ListAllProcessesByQuery(query url.Values) ([]Process, error)
	ListServicePlanVisibilitiesByQuery(query url.Values) ([]ServicePlanVisibility, error)
	ListServicePlanVisibilities() ([]ServicePlanVisibility, error)
	GetServicePlanVisibilityByGuid(guid string) (ServicePlanVisibility, error)
	CreateServicePlanVisibilityByUniqueId(uniqueId string, organizationGuid string) (ServicePlanVisibility, error)
	CreateServicePlanVisibility(servicePlanGuid string, organizationGuid string) (ServicePlanVisibility, error)
	DeleteServicePlanVisibilityByPlanAndOrg(servicePlanGuid string, organizationGuid string, async bool) error
	DeleteServicePlanVisibility(guid string, async bool) error
	UpdateServicePlanVisibility(guid string, servicePlanGuid string, organizationGuid string) (ServicePlanVisibility, error)
	ListStacksByQuery(query url.Values) ([]Stack, error)
	ListStacks() ([]Stack, error)
	GetRunningEnvironmentVariableGroup() (EnvironmentVariableGroup, error)
	GetStagingEnvironmentVariableGroup() (EnvironmentVariableGroup, error)
	SetRunningEnvironmentVariableGroup(evg EnvironmentVariableGroup) error
	SetStagingEnvironmentVariableGroup(evg EnvironmentVariableGroup) error
	ListSecGroups() (secGroups []SecGroup, err error)
	ListRunningSecGroups() ([]SecGroup, error)
	ListStagingSecGroups() ([]SecGroup, error)
	GetSecGroupByName(name string) (secGroup SecGroup, err error)
	CreateSecGroup(name string, rules []SecGroupRule, spaceGuids []string) (*SecGroup, error)
	UpdateSecGroup(guid, name string, rules []SecGroupRule, spaceGuids []string) (*SecGroup, error)
	DeleteSecGroup(guid string) error
	GetSecGroup(guid string) (*SecGroup, error)
	BindSecGroup(secGUID, spaceGUID string) error
	BindStagingSecGroupToSpace(secGUID, spaceGUID string) error
	BindRunningSecGroup(secGUID string) error
	UnbindRunningSecGroup(secGUID string) error
	BindStagingSecGroup(secGUID string) error
	UnbindStagingSecGroup(secGUID string) error
	UnbindSecGroup(secGUID, spaceGUID string) error
	CreateIsolationSegment(name string) (*IsolationSegment, error)
	GetIsolationSegmentByGUID(guid string) (*IsolationSegment, error)
	ListIsolationSegmentsByQuery(query url.Values) ([]IsolationSegment, error)
	ListIsolationSegments() ([]IsolationSegment, error)
	DeleteIsolationSegmentByGUID(guid string) error
	AddIsolationSegmentToOrg(isolationSegmentGUID, orgGUID string) error
	RemoveIsolationSegmentFromOrg(isolationSegmentGUID, orgGUID string) error
	AddIsolationSegmentToSpace(isolationSegmentGUID, spaceGUID string) error
	RemoveIsolationSegmentFromSpace(isolationSegmentGUID, spaceGUID string) error
	ListAppEvents(eventType string) ([]AppEventEntity, error)
	ListAppEventsByQuery(eventType string, queries []AppEventQuery) ([]AppEventEntity, error)
	GetInfo() (*Info, error)
	CreateBuildpack(bpr *BuildpackRequest) (*Buildpack, error)
	ListBuildpacks() ([]Buildpack, error)
	DeleteBuildpack(guid string, async bool) error
	GetBuildpackByGuid(buildpackGUID string) (Buildpack, error)
	CreateSpace(req SpaceRequest) (Space, error)
	UpdateSpace(spaceGUID string, req SpaceRequest) (Space, error)
	DeleteSpace(guid string, recursive, async bool) error
	ListSpaceManagersByQuery(spaceGUID string, query url.Values) ([]User, error)
	ListSpaceManagers(spaceGUID string) ([]User, error)
	ListSpaceAuditorsByQuery(spaceGUID string, query url.Values) ([]User, error)
	ListSpaceAuditors(spaceGUID string) ([]User, error)
	ListSpaceDevelopersByQuery(spaceGUID string, query url.Values) ([]User, error)
	ListSpaceDevelopers(spaceGUID string) ([]User, error)
	AssociateSpaceDeveloper(spaceGUID, userGUID string) (Space, error)
	AssociateSpaceDeveloperByUsername(spaceGUID, name string) (Space, error)
	AssociateSpaceDeveloperByUsernameAndOrigin(spaceGUID, name, origin string) (Space, error)
	RemoveSpaceDeveloper(spaceGUID, userGUID string) error
	RemoveSpaceDeveloperByUsername(spaceGUID, name string) error
	RemoveSpaceDeveloperByUsernameAndOrigin(spaceGUID, name, origin string) error
	AssociateSpaceAuditor(spaceGUID, userGUID string) (Space, error)
	AssociateSpaceAuditorByUsername(spaceGUID, name string) (Space, error)
	AssociateSpaceAuditorByUsernameAndOrigin(spaceGUID, name, origin string) (Space, error)
	RemoveSpaceAuditor(spaceGUID, userGUID string) error
	RemoveSpaceAuditorByUsername(spaceGUID, name string) error
	RemoveSpaceAuditorByUsernameAndOrigin(spaceGUID, name, origin string) error
	AssociateSpaceManager(spaceGUID, userGUID string) (Space, error)
	AssociateSpaceManagerByUsername(spaceGUID, name string) (Space, error)
	AssociateSpaceManagerByUsernameAndOrigin(spaceGUID, name, origin string) (Space, error)
	RemoveSpaceManager(spaceGUID, userGUID string) error
	RemoveSpaceManagerByUsername(spaceGUID, name string) error
	RemoveSpaceManagerByUsernameAndOrigin(spaceGUID, name, origin string) error
	ListSpaceSecGroups(spaceGUID string) (secGroups []SecGroup, err error)
	ListSpacesByQuery(query url.Values) ([]Space, error)
	ListSpaces() ([]Space, error)
	GetSpaceByName(spaceName string, orgGuid string) (space Space, err error)
	GetSpaceByGuid(spaceGUID string) (Space, error)
	IsolationSegmentForSpace(spaceGUID, isolationSegmentGUID string) error
	ResetIsolationSegmentForSpace(spaceGUID string) error
	ListDomainsByQuery(query url.Values) ([]Domain, error)
	ListDomains() ([]Domain, error)
	ListSharedDomainsByQuery(query url.Values) ([]SharedDomain, error)
	ListSharedDomains() ([]SharedDomain, error)
	GetSharedDomainByGuid(guid string) (SharedDomain, error)
	CreateSharedDomain(name string, internal bool, router_group_guid string) (*SharedDomain, error)
	DeleteSharedDomain(guid string, async bool) error
	GetDomainByName(name string) (Domain, error)
	GetSharedDomainByName(name string) (SharedDomain, error)
	CreateDomain(name, orgGuid string) (*Domain, error)
	DeleteDomain(guid string) error
	DeleteServiceBroker(guid string) error
	UpdateServiceBroker(guid string, usb UpdateServiceBrokerRequest) (ServiceBroker, error)
	CreateServiceBroker(csb CreateServiceBrokerRequest) (ServiceBroker, error)
	ListServiceBrokersByQuery(query url.Values) ([]ServiceBroker, error)
	ListServiceBrokers() ([]ServiceBroker, error)
	GetServiceBrokerByGuid(guid string) (ServiceBroker, error)
	GetServiceBrokerByName(name string) (ServiceBroker, error)
	ListServiceUsageEventsByQuery(query url.Values) ([]ServiceUsageEvent, error)
	ListServiceUsageEvents() ([]ServiceUsageEvent, error)
	ListServiceKeysByQuery(query url.Values) ([]ServiceKey, error)
	ListServiceKeys() ([]ServiceKey, error)
	GetServiceKeyByName(name string) (ServiceKey, error)
	GetServiceKeyByInstanceGuid(guid string) (ServiceKey, error)
	GetServiceKeysByInstanceGuid(guid string) ([]ServiceKey, error)
	CreateServiceKey(csr CreateServiceKeyRequest) (ServiceKey, error)
	DeleteServiceKey(guid string) error
	ListOrgsByQuery(query url.Values) ([]Org, error)
	ListOrgs() ([]Org, error)
	GetOrgByName(name string) (Org, error)
	GetOrgByGuid(guid string) (Org, error)
	OrgSpaces(guid string) ([]Space, error)
	ListOrgUsersByQuery(orgGUID string, query url.Values) ([]User, error)
	ListOrgUsers(orgGUID string) ([]User, error)
	ListOrgManagersByQuery(orgGUID string, query url.Values) ([]User, error)
	ListOrgManagers(orgGUID string) ([]User, error)
	ListOrgAuditorsByQuery(orgGUID string, query url.Values) ([]User, error)
	ListOrgAuditors(orgGUID string) ([]User, error)
	ListOrgBillingManagersByQuery(orgGUID string, query url.Values) ([]User, error)
	ListOrgBillingManagers(orgGUID string) ([]User, error)
	AssociateOrgManager(orgGUID, userGUID string) (Org, error)
	AssociateOrgManagerByUsername(orgGUID, name string) (Org, error)
	AssociateOrgManagerByUsernameAndOrigin(orgGUID, name, origin string) (Org, error)
	AssociateOrgUser(orgGUID, userGUID string) (Org, error)
	AssociateOrgAuditor(orgGUID, userGUID string) (Org, error)
	AssociateOrgUserByUsername(orgGUID, name string) (Org, error)
	AssociateOrgUserByUsernameAndOrigin(orgGUID, name, origin string) (Org, error)
	AssociateOrgAuditorByUsername(orgGUID, name string) (Org, error)
	AssociateOrgAuditorByUsernameAndOrigin(orgGUID, name, origin string) (Org, error)
	AssociateOrgBillingManager(orgGUID, userGUID string) (Org, error)
	AssociateOrgBillingManagerByUsername(orgGUID, name string) (Org, error)
	AssociateOrgBillingManagerByUsernameAndOrigin(orgGUID, name, origin string) (Org, error)
	RemoveOrgManager(orgGUID, userGUID string) error
	RemoveOrgManagerByUsername(orgGUID, name string) error
	RemoveOrgManagerByUsernameAndOrigin(orgGUID, name, origin string) error
	RemoveOrgUser(orgGUID, userGUID string) error
	RemoveOrgAuditor(orgGUID, userGUID string) error
	RemoveOrgUserByUsername(orgGUID, name string) error
	RemoveOrgUserByUsernameAndOrigin(orgGUID, name, origin string) error
	RemoveOrgAuditorByUsername(orgGUID, name string) error
	RemoveOrgAuditorByUsernameAndOrigin(orgGUID, name, origin string) error
	RemoveOrgBillingManager(orgGUID, userGUID string) error
	RemoveOrgBillingManagerByUsername(orgGUID, name string) error
	RemoveOrgBillingManagerByUsernameAndOrigin(orgGUID, name, origin string) error
	ListOrgSpaceQuotas(orgGUID string) ([]SpaceQuota, error)
	ListOrgPrivateDomains(orgGUID string) ([]Domain, error)
	ShareOrgPrivateDomain(orgGUID, privateDomainGUID string) (*Domain, error)
	UnshareOrgPrivateDomain(orgGUID, privateDomainGUID string) error
	CreateOrg(req OrgRequest) (Org, error)
	UpdateOrg(orgGUID string, orgRequest OrgRequest) (Org, error)
	DeleteOrg(guid string, recursive, async bool) error
	DefaultIsolationSegmentForOrg(orgGUID, isolationSegmentGUID string) error
	ResetDefaultIsolationSegmentForOrg(orgGUID string) error
	ListEventsByQuery(query url.Values) ([]Event, error)
	ListEvents() ([]Event, error)
	TotalEventsByQuery(query url.Values) (int, error)
	TotalEvents() (int, error)
	GetUserByGUID(guid string) (User, error)
	ListUsersByQuery(query url.Values) (Users, error)
	ListUsers() (Users, error)
	ListUserSpaces(userGuid string) ([]Space, error)
	ListUserAuditedSpaces(userGuid string) ([]Space, error)
	ListUserManagedSpaces(userGuid string) ([]Space, error)
	ListUserOrgs(userGuid string) ([]Org, error)
	ListUserManagedOrgs(userGuid string) ([]Org, error)
	ListUserAuditedOrgs(userGuid string) ([]Org, error)
	ListUserBillingManagedOrgs(userGuid string) ([]Org, error)
	CreateUser(req UserRequest) (User, error)
	DeleteUser(userGuid string) error
	ListServicePlansByQuery(query url.Values) ([]ServicePlan, error)
	ListServicePlans() ([]ServicePlan, error)
	GetServicePlanByGUID(guid string) (*ServicePlan, error)
	MakeServicePlanPublic(servicePlanGUID string) error
	MakeServicePlanPrivate(servicePlanGUID string) error
	ListServiceBindingsByQuery(query url.Values) ([]ServiceBinding, error)
	ListServiceBindings() ([]ServiceBinding, error)
	GetServiceBindingByGuid(guid string) (ServiceBinding, error)
	ServiceBindingByGuid(guid string) (ServiceBinding, error)
	DeleteServiceBinding(guid string) error
	CreateServiceBinding(appGUID, serviceInstanceGUID string) (*ServiceBinding, error)
	CreateRouteServiceBinding(routeGUID, serviceInstanceGUID string) error
	DeleteRouteServiceBinding(routeGUID, serviceInstanceGUID string) error
	UpdateApp(guid string, aur AppUpdateResource) (UpdateResponse, error)
	ListServiceInstancesByQuery(query url.Values) ([]ServiceInstance, error)
	ListServiceInstances() ([]ServiceInstance, error)
	GetServiceInstanceByGuid(guid string) (ServiceInstance, error)
	ServiceInstanceByGuid(guid string) (ServiceInstance, error)
	CreateServiceInstance(req ServiceInstanceRequest) (ServiceInstance, error)
	UpdateServiceInstance(serviceInstanceGuid string, updatedConfiguration io.Reader, async bool) error
	DeleteServiceInstance(guid string, recursive, async bool) error
	ListOrgQuotasByQuery(query url.Values) ([]OrgQuota, error)
	ListOrgQuotas() ([]OrgQuota, error)
	GetOrgQuotaByName(name string) (OrgQuota, error)
	CreateOrgQuota(orgQuote OrgQuotaRequest) (*OrgQuota, error)
	UpdateOrgQuota(orgQuotaGUID string, orgQuota OrgQuotaRequest) (*OrgQuota, error)
	DeleteOrgQuota(guid string, async bool) error
	CreateRoute(routeRequest RouteRequest) (Route, error)
	CreateTcpRoute(routeRequest RouteRequest) (Route, error)
	BindRoute(routeGUID, appGUID string) error
	ListRoutesByQuery(query url.Values) ([]Route, error)
	ListRoutes() ([]Route, error)
	DeleteRoute(guid string) error
	ListTasks() ([]Task, error)
	ListTasksByQuery(query url.Values) ([]Task, error)
	TasksByApp(guid string) ([]Task, error)
	TasksByAppByQuery(guid string, query url.Values) ([]Task, error)
	CreateTask(tr TaskRequest) (task Task, err error)
	GetTaskByGuid(guid string) (task Task, err error)
	TaskByGuid(guid string) (task Task, err error)
	TerminateTask(guid string) error
	MappingAppAndRoute(req RouteMappingRequest) (*RouteMapping, error)
	ListRouteMappings() ([]*RouteMapping, error)
	ListRouteMappingsByQuery(query url.Values) ([]*RouteMapping, error)
	GetRouteMappingByGuid(guid string) (*RouteMapping, error)
	DeleteRouteMapping(guid string) error
	ListAppUsageEventsByQuery(query url.Values) ([]AppUsageEvent, error)
	ListAppUsageEvents() ([]AppUsageEvent, error)
	ListAppsByQueryWithLimits(query url.Values, totalPages int) ([]App, error)
	ListAppsByQuery(query url.Values) ([]App, error)
	GetAppByGuidNoInlineCall(guid string) (App, error)
	ListApps() ([]App, error)
	ListAppsByRoute(routeGuid string) ([]App, error)
	GetAppInstances(guid string) (map[string]AppInstance, error)
	GetAppEnv(guid string) (AppEnv, error)
	GetAppRoutes(guid string) ([]Route, error)
	GetAppStats(guid string) (map[string]AppStats, error)
	KillAppInstance(guid string, index string) error
	GetAppByGuid(guid string) (App, error)
	AppByGuid(guid string) (App, error)
	AppByName(appName, spaceGuid, orgGuid string) (app App, err error)
	UploadAppBits(file io.Reader, appGUID string) error
	GetAppBits(guid string) (io.ReadCloser, error)
	CreateApp(req AppCreateRequest) (App, error)
	StartApp(guid string) error
	StopApp(guid string) error
	GetServiceByGuid(guid string) (Service, error)
	ListServicesByQuery(query url.Values) ([]Service, error)
	ListServices() ([]Service, error)
	NewRequest(method, path string) *Request
	NewRequestWithBody(method, path string, body io.Reader) *Request
	DoRequest(r *Request) (*http.Response, error)
	DoRequestWithoutRedirects(r *Request) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
	GetToken() (string, error)
	ListUserProvidedServiceInstancesByQuery(query url.Values) ([]UserProvidedServiceInstance, error)
	ListUserProvidedServiceInstances() ([]UserProvidedServiceInstance, error)
	GetUserProvidedServiceInstanceByGuid(guid string) (UserProvidedServiceInstance, error)
	UserProvidedServiceInstanceByGuid(guid string) (UserProvidedServiceInstance, error)
	CreateUserProvidedServiceInstance(req UserProvidedServiceInstanceRequest) (*UserProvidedServiceInstance, error)
	DeleteUserProvidedServiceInstance(guid string) error
	UpdateUserProvidedServiceInstance(guid string, req UserProvidedServiceInstanceRequest) (*UserProvidedServiceInstance, error)
	ListSpaceQuotasByQuery(query url.Values) ([]SpaceQuota, error)
	ListSpaceQuotas() ([]SpaceQuota, error)
	GetSpaceQuotaByName(name string) (SpaceQuota, error)
	AssignSpaceQuota(quotaGUID, spaceGUID string) error
	CreateSpaceQuota(spaceQuote SpaceQuotaRequest) (*SpaceQuota, error)
	UpdateSpaceQuota(spaceQuotaGUID string, spaceQuote SpaceQuotaRequest) (*SpaceQuota, error)
}
