---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-azure.html
---

# Azure fields [exported-fields-azure]

azure module

**`azure.timegrain`**
:   The Azure metric timegrain

type: keyword



## resource [_resource]

The resource specified

**`azure.resource.type`**
:   The type of the resource

type: keyword


**`azure.resource.name`**
:   The name of the resource

type: keyword


**`azure.resource.group`**
:   The resource group

type: keyword


**`azure.resource.tags.*`**
:   Azure resource tags.

type: object


**`azure.namespace`**
:   The namespace selected

type: keyword


**`azure.subscription_id`**
:   The subscription ID

type: keyword


**`azure.subscription_name`**
:   The subscription name

type: keyword


**`azure.application_id`**
:   The application ID

type: keyword


**`azure.dimensions.*`**
:   Azure metric dimensions.

type: object


**`azure.metrics.*.*`**
:   Metrics returned.

type: object



## app_insights [_app_insights_2]

application insights

**`azure.app_insights.start_date`**
:   The start date

type: date


**`azure.app_insights.end_date`**
:   The end date

type: date


**`azure.app_insights.metrics.*.*`**
:   The metrics

type: object



## app_state [_app_state_2]

application state

**`azure.app_state.start_date`**
:   The start date

type: date


**`azure.app_state.end_date`**
:   The end date

type: date


**`azure.app_state.requests_count.sum`**
:   Request count

type: float


**`azure.app_state.requests_failed.sum`**
:   Request failed count

type: float


**`azure.app_state.users_count.unique`**
:   User count

type: float


**`azure.app_state.sessions_count.unique`**
:   Session count

type: float


**`azure.app_state.users_authenticated.unique`**
:   Authenticated users count

type: float


**`azure.app_state.browser_timings_network_duration.avg`**
:   Browser timings network duration

type: float


**`azure.app_state.browser_timings_send_duration.avg`**
:   Browser timings send duration

type: float


**`azure.app_state.browser_timings_receive_uration.avg`**
:   Browser timings receive duration

type: float


**`azure.app_state.browser_timings_processing_duration.avg`**
:   Browser timings processing duration

type: float


**`azure.app_state.browser_timings_total_duration.avg`**
:   Browser timings total duration

type: float


**`azure.app_state.exceptions_count.sum`**
:   Exception count

type: float


**`azure.app_state.exceptions_browser.sum`**
:   Exception count at browser level

type: float


**`azure.app_state.exceptions_server.sum`**
:   Exception count at server level

type: float


**`azure.app_state.performance_counters_memory_available_bytes.avg`**
:   Performance counters memory available bytes

type: float


**`azure.app_state.performance_counters_process_private_bytes.avg`**
:   Performance counters process private bytes

type: float


**`azure.app_state.performance_counters_process_cpu_percentage_total.avg`**
:   Performance counters process cpu percentage total

type: float


**`azure.app_state.performance_counters_process_cpu_percentage.avg`**
:   Performance counters process cpu percentage

type: float


**`azure.app_state.performance_counters_processiobytes_per_second.avg`**
:   Performance counters process IO bytes per second

type: float



## billing [_billing_5]

billing and usage details

**`azure.billing.currency`**
:   Billing Currency.

type: keyword


**`azure.billing.pretax_cost`**
:   The amount of cost before tax.

type: float


**`azure.billing.unit_price`**
:   Unit Price is the price applicable to you. (your EA or other contract price).

type: float


**`azure.billing.quantity`**
:   Measure the quantity purchased or consumed. The amount of the meter used during the billing period.

type: float


**`azure.billing.department_name`**
:   The department name

type: keyword


**`azure.billing.product`**
:   Product name for the consumed service or purchase.

type: keyword


**`azure.billing.usage_start`**
:   The usage start date

type: date


**`azure.billing.usage_end`**
:   The usage end date

type: date


**`azure.billing.billing_period_id`**
:   The billing period id.

type: keyword


**`azure.billing.account_name`**
:   Name of the Billing Account.

type: keyword


**`azure.billing.account_id`**
:   Billing Account identifier.

type: keyword


**`azure.billing.actual_cost`**
:   The actual cost

type: float


**`azure.billing.forecast_cost`**
:   The forecast cost

type: float


**`azure.billing.usage_date`**
:   The usage date

type: date


**`azure.compute_vm.*.*`**
:   compute_vm

type: object


**`azure.compute_vm_scaleset.*.*`**
:   compute_vm_scaleset

type: object


**`azure.container_instance.*.*`**
:   container instance

type: object


**`azure.container_registry.*.*`**
:   container registry

type: object


**`azure.container_service.*.*`**
:   container service

type: object


**`azure.database_account.*.*`**
:   database account

type: object



## monitor [_monitor_2]

monitor

**`azure.storage.*.*`**
:   storage account

type: object


**`azure.storage_account.*.*`**
:   storage account

type: object


