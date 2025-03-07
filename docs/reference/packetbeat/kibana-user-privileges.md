---
navigation_title: "Create a _reader_ user"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/kibana-user-privileges.html
---

# Grant privileges and roles needed to read Packetbeat data from {{kib}} [kibana-user-privileges]


{{kib}} users typically need to view dashboards and visualizations that contain Packetbeat data. These users might also need to create and edit dashboards and visualizations.

To grant users the required privileges:

1. Create a **reader role**, called something like `packetbeat_reader`, that has the following privilege:

    | Type | Privilege | Purpose |
    | --- | --- | --- |
    | Index | `read` on `packetbeat-*` indices | Read data indexed by Packetbeat |
    | Spaces | `Read` or `All` on Dashboards, Visualize, and Discover | Allow the user to view, edit, and create dashboards, as well as browse data. |

2. Assign the **reader role**, along with the following built-in roles, to users who need to read Packetbeat data:

    | Role | Purpose |
    | --- | --- |
    | `monitoring_user` | Allow users to monitor the health of Packetbeat itself. Only assign this role to users who manage Packetbeat. |


