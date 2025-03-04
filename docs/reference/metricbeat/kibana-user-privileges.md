---
navigation_title: "Create a _reader_ user"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/kibana-user-privileges.html
---

# Grant privileges and roles needed to read Metricbeat data from {{kib}} [kibana-user-privileges]


{{kib}} users typically need to view dashboards and visualizations that contain Metricbeat data. These users might also need to create and edit dashboards and visualizations.

To grant users the required privileges:

1. Create a **reader role**, called something like `metricbeat_reader`, that has the following privilege:

    | Type | Privilege | Purpose |
    | --- | --- | --- |
    | Index | `read` on `metricbeat-*` indices | Read data indexed by Metricbeat |
    | Spaces | `Read` or `All` on Dashboards, Visualize, and Discover | Allow the user to view, edit, and create dashboards, as well as browse data. |
    | Spaces | `Read` or `All` on {{kib}} Metrics | Allow the use of {{kib}} Metrics |

2. Assign the **reader role**, along with the following built-in roles, to users who need to read Metricbeat data:

    | Role | Purpose |
    | --- | --- |
    | `monitoring_user` | Allow users to monitor the health of Metricbeat itself. Only assign this role to users who manage Metricbeat. |


