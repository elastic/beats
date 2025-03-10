---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-istio.html
---

# Istio fields [exported-fields-istio]

istio Module


## istio [_istio]

`istio` contains statistics that were read from Istio


## citadel [_citadel]

Contains statistics related to the Istio Citadel service

**`istio.citadel.grpc.method`**
:   The grpc method

type: keyword


**`istio.citadel.grpc.service`**
:   The grpc service

type: keyword


**`istio.citadel.grpc.type`**
:   The type of the respective grpc service

type: keyword


**`istio.citadel.secret_controller_svc_acc_created_cert.count`**
:   The number of certificates created due to service account creation.

type: long


**`istio.citadel.server_root_cert_expiry_seconds`**
:   The unix timestamp, in seconds, when Citadel root cert will expire. We set it to negative in case of internal error.

type: float


**`istio.citadel.grpc.server.handled`**
:   Total number of RPCs completed on the server, regardless of success or failure.

type: long


**`istio.citadel.grpc.server.msg.received`**
:   Total number of RPC stream messages received on the server.

type: long


**`istio.citadel.grpc.server.msg.sent`**
:   Total number of gRPC stream messages sent by the server.

type: long


**`istio.citadel.grpc.server.started`**
:   Total number of RPCs started on the server.

type: long


**`istio.citadel.grpc.server.handling.latency.ms.bucket.*`**
:   The response latency (milliseconds) of gRPC that had been application-level handled by the server.

type: object


**`istio.citadel.grpc.server.handling.latency.ms.sum`**
:   The response latency of gRPC, sum of latencies in milliseconds

type: long

format: duration


**`istio.citadel.grpc.server.handling.latency.ms.count`**
:   The response latency of gRPC, number of metrics

type: long



## galley [_galley]

Contains statistics related to the Istio galley service

**`istio.galley.name`**
:   The name of the resource the metric is related to

type: keyword


**`istio.galley.namespace`**
:   The Kubernetes namespace of the resource

type: keyword


**`istio.galley.version`**
:   The version of the object

type: keyword


**`istio.galley.collection`**
:   The collection of the instance

type: keyword


**`istio.galley.istio.authentication.meshpolicies`**
:   The number of valid istio/authentication/meshpolicies known to galley at a point in time

type: long


**`istio.galley.istio.authentication.policies`**
:   The number of valid istio/authentication/policies known to galley at a point in time

type: long


**`istio.galley.istio.mesh.MeshConfig`**
:   The number of valid istio/mesh/MeshConfig known to galley at a point in time

type: long


**`istio.galley.istio.networking.destinationrules`**
:   The number of valid istio/networking/destinationrules known to galley at a point in time

type: long


**`istio.galley.istio.networking.envoyfilters`**
:   The number of valid istio/networking/envoyfilters known to galley at a point in time

type: long


**`istio.galley.istio.networking.gateways`**
:   The number of valid istio/networking/gateways known to galley at a point in time

type: long


**`istio.galley.istio.networking.sidecars`**
:   The number of valid istio/networking/sidecars known to galley at a point in time

type: long


**`istio.galley.istio.networking.virtualservices`**
:   The number of valid istio/networking/virtualservices known to galley at a point in time

type: long


**`istio.galley.istio.policy.attributemanifests`**
:   The number of valid istio/policy/attributemanifests known to galley at a point in time

type: long


**`istio.galley.istio.policy.handlers`**
:   The number of valid istio/policy/handlers known to galley at a point in time

type: long


**`istio.galley.istio.policy.instances`**
:   The number of valid istio/policy/instances known to galley at a point in time

type: long


**`istio.galley.istio.policy.rules`**
:   The number of valid istio/policy/rules known to galley at a point in time

type: long


**`istio.galley.runtime.processor.event_span.duration.ms.bucket.*`**
:   The duration between each incoming event as histogram buckets in milliseconds

type: object


**`istio.galley.runtime.processor.event_span.duration.ms.sum`**
:   The duration between each incoming event, sum of durations in milliseconds

type: long

format: duration


**`istio.galley.runtime.processor.event_span.duration.ms.count`**
:   The duration between each incoming event, number of metrics

type: long


**`istio.galley.runtime.processor.snapshot_events.bucket.*`**
:   The number of events that have been processed as histogram buckets

type: object


**`istio.galley.runtime.processor.snapshot_events.sum`**
:   The number of events that have been processed, sum of events

type: long


**`istio.galley.runtime.processor.snapshot_events.count`**
:   The duration between each incoming event, number of metrics

type: long


**`istio.galley.runtime.processor.snapshot_lifetime.duration.ms.bucket.*`**
:   The duration of each snapshot as histogram buckets in milliseconds

type: object


**`istio.galley.runtime.processor.snapshot_lifetime.duration.ms.sum`**
:   The duration of each snapshot, sum of durations in milliseconds

type: long

format: duration


**`istio.galley.runtime.processor.snapshot_lifetime.duration.ms.count`**
:   The duration of each snapshot, number of metrics

type: long


**`istio.galley.runtime.state_type_instances`**
:   The number of type instances per type URL

type: long


**`istio.galley.runtime.strategy.on_change`**
:   The number of times the strategy’s onChange has been called

type: long


**`istio.galley.runtime.strategy.timer_quiesce_reached`**
:   The number of times a quiesce has been reached

type: long


**`istio.galley.source_kube_event_success_total`**
:   The number of times a kubernetes source successfully handled an event

type: long


**`istio.galley.validation.cert_key.updates`**
:   Galley validation webhook certificate updates

type: long


**`istio.galley.validation.config.load`**
:   k8s webhook configuration (re)loads

type: long


**`istio.galley.validation.config.updates`**
:   k8s webhook configuration updates

type: long



## mesh [_mesh]

Contains statistics related to the Istio mesh service

**`istio.mesh.instance`**
:   The prometheus instance

type: text


**`istio.mesh.job`**
:   The prometheus job

type: keyword


**`istio.mesh.requests`**
:   Total requests handled by an Istio proxy

type: long


**`istio.mesh.request.duration.ms.bucket.*`**
:   Request duration histogram buckets in milliseconds

type: object


**`istio.mesh.request.duration.ms.sum`**
:   Requests duration, sum of durations in milliseconds

type: long

format: duration


**`istio.mesh.request.duration.ms.count`**
:   Requests duration, number of requests

type: long


**`istio.mesh.request.size.bytes.bucket.*`**
:   Request Size histogram buckets

type: object


**`istio.mesh.request.size.bytes.sum`**
:   Request Size histogram sum

type: long


**`istio.mesh.request.size.bytes.count`**
:   Request Size histogram count

type: long


**`istio.mesh.response.size.bytes.bucket.*`**
:   Request Size histogram buckets

type: object


**`istio.mesh.response.size.bytes.sum`**
:   Request Size histogram sum

type: long


**`istio.mesh.response.size.bytes.count`**
:   Request Size histogram count

type: long


**`istio.mesh.reporter`**
:   Reporter identifies the reporter of the request. It is set to destination if report is from a server Istio proxy and source if report is from a client Istio proxy.

type: keyword


**`istio.mesh.source.workload.name`**
:   This identifies the name of source workload which controls the source.

type: keyword


**`istio.mesh.source.workload.namespace`**
:   This identifies the namespace of the source workload.

type: keyword


**`istio.mesh.source.principal`**
:   This identifies the peer principal of the traffic source. It is set when peer authentication is used.

type: keyword


**`istio.mesh.source.app`**
:   This identifies the source app based on app label of the source workload.

type: keyword


**`istio.mesh.source.version`**
:   This identifies the version of the source workload.

type: keyword


**`istio.mesh.destination.workload.name`**
:   This identifies the name of destination workload.

type: keyword


**`istio.mesh.destination.workload.namespace`**
:   This identifies the namespace of the destination workload.

type: keyword


**`istio.mesh.destination.principal`**
:   This identifies the peer principal of the traffic destination. It is set when peer authentication is used.

type: keyword


**`istio.mesh.destination.app`**
:   This identifies the destination app based on app label of the destination workload..

type: keyword


**`istio.mesh.destination.version`**
:   This identifies the version of the destination workload.

type: keyword


**`istio.mesh.destination.service.host`**
:   This identifies destination service host responsible for an incoming request.

type: keyword


**`istio.mesh.destination.service.name`**
:   This identifies the destination service name.

type: keyword


**`istio.mesh.destination.service.namespace`**
:   This identifies the namespace of destination service.

type: keyword


**`istio.mesh.request.protocol`**
:   This identifies the protocol of the request. It is set to API protocol if provided, otherwise request or connection protocol.

type: keyword


**`istio.mesh.response.code`**
:   This identifies the response code of the request. This label is present only on HTTP metrics.

type: long


**`istio.mesh.connection.security.policy`**
:   This identifies the service authentication policy of the request. It is set to mutual_tls when Istio is used to make communication secure and report is from destination. It is set to unknown when report is from source since security policy cannot be properly populated.

type: keyword



## mixer [_mixer]

Contains statistics related to the Istio mixer service

**`istio.mixer.istio.mcp.request.acks`**
:   The number of request acks received by the source.

type: long


**`istio.mixer.config.adapter.info.errors.config`**
:   The number of errors encountered during processing of the adapter info configuration.

type: long


**`istio.mixer.config.adapter.info.configs`**
:   The number of known adapters in the current config.

type: long


**`istio.mixer.config.attributes`**
:   The number of known attributes in the current config.

type: long


**`istio.mixer.config.handler.configs`**
:   The number of known handlers in the current config.

type: long


**`istio.mixer.config.handler.errors.validation`**
:   The number of errors encountered because handler validation returned error.

type: long


**`istio.mixer.config.instance.errors.config`**
:   The number of errors encountered during processing of the instance configuration.

type: long


**`istio.mixer.config.instance.configs`**
:   The number of known instances in the current config.

type: long


**`istio.mixer.config.rule.errors.config`**
:   The number of errors encountered during processing of the rule configuration.

type: long


**`istio.mixer.config.rule.errors.match`**
:   The number of rule conditions that was not parseable.

type: long


**`istio.mixer.config.rule.configs`**
:   The number of known rules in the current config.

type: long


**`istio.mixer.config.template.errors.config`**
:   The number of errors encountered during processing of the template configuration.

type: long


**`istio.mixer.config.template.configs`**
:   The number of known templates in the current config.

type: long


**`istio.mixer.config.unsatisfied.action_handler`**
:   The number of actions that failed due to handlers being unavailable.

type: long


**`istio.mixer.dispatcher_destinations_per_variety_total`**
:   The number of Mixer adapter destinations by template variety type.

type: long


**`istio.mixer.handler.handlers.closed`**
:   The number of handlers that were closed during config transition.

type: long


**`istio.mixer.handler.daemons`**
:   The current number of active daemon routines in a given adapter environment.

type: long


**`istio.mixer.handler.failures.build`**
:   The number of handlers that failed creation during config transition.

type: long


**`istio.mixer.handler.failures.close`**
:   The number of errors encountered while closing handlers during config transition.

type: long


**`istio.mixer.handler.handlers.new`**
:   The number of handlers that were newly created during config transition.

type: long


**`istio.mixer.handler.handlers.reused`**
:   The number of handlers that were re-used during config transition.

type: long


**`istio.mixer.handler.name`**
:   The name of the daemon  handler

type: keyword


**`istio.mixer.variety`**
:   The name of the variety

type: keyword



## pilot [_pilot]

Contains statistics related to the Istio pilot service

**`istio.pilot.xds.count`**
:   Count of concurrent xDS client connections for Pilot.

type: long


**`istio.pilot.xds.pushes`**
:   Count of xDS messages sent, as well as errors building or sending xDS messages for lds, rds, cds and eds.

type: long


**`istio.pilot.xds.push.time.ms.bucket.*`**
:   Total time Pilot takes to push lds, rds, cds and eds, histogram buckets in milliseconds.

type: object


**`istio.pilot.xds.push.time.ms.sum`**
:   Total time Pilot takes to push lds, rds, cds and eds, histogram sum of times in milliseconds.

type: long


**`istio.pilot.xds.push.time.ms.count`**
:   Total time Pilot takes to push lds, rds, cds and eds, histogram count of times.

type: long


**`istio.pilot.xds.eds.instances`**
:   Instances for each cluster, as of last push. Zero instances is an error.

type: long


**`istio.pilot.xds.push.context.errors`**
:   Number of errors (timeouts) initiating push context.

type: long


**`istio.pilot.xds.internal.errors`**
:   Total number of internal XDS errors in pilot.

type: long


**`istio.pilot.conflict.listener.inbound`**
:   Number of conflicting inbound listeners.

type: long


**`istio.pilot.conflict.listener.outbound.http.over.current.tcp`**
:   Number of conflicting wildcard http listeners with current wildcard tcp listener.

type: long


**`istio.pilot.conflict.listener.outbound.http.over.https`**
:   Number of conflicting HTTP listeners with well known HTTPS ports.

type: long


**`istio.pilot.conflict.listener.outbound.tcp.over.current.http`**
:   Number of conflicting wildcard tcp listeners with current wildcard http listener.

type: long


**`istio.pilot.conflict.listener.outbound.tcp.over.current.tcp`**
:   Number of conflicting tcp listeners with current tcp listener.

type: long


**`istio.pilot.proxy.conv.ms.bucket.*`**
:   Time needed by Pilot to push Envoy configurations, histogram buckets in milliseconds.

type: object


**`istio.pilot.proxy.conv.ms.sum`**
:   Time needed by Pilot to push Envoy configurations, histogram sum of times in milliseconds.

type: long


**`istio.pilot.proxy.conv.ms.count`**
:   Time needed by Pilot to push Envoy configurations, histogram count of times.

type: long


**`istio.pilot.services`**
:   Total services known to pilot.

type: integer


**`istio.pilot.virt.services`**
:   Total virtual services known to pilot.

type: long


**`istio.pilot.no.ip`**
:   Pods not found in the endpoint table, possibly invalid.

type: long


**`istio.pilot.cluster`**
:   The instance FQDN.

type: text


**`istio.pilot.type`**
:   The Envoy proxy configuration type.

type: text


