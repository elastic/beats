::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the `user` dataset of the system module.

It is implemented for Linux only.


## Required privileges [_required_privileges_user]

The `user` dataset reads `/etc/passwd` and `/etc/group`, which are world-readable on all standard Linux distributions. No elevated privileges are required for basic operation.

When `user.detect_password_changes: true` is set, the dataset also reads `/etc/shadow` to detect password hash changes. The shadow file is readable only by root, or by the `shadow` group on some distributions.

| Configuration | Minimum privilege |
|---|---|
| `detect_password_changes: false` (default) | None — any user can run this dataset |
| `detect_password_changes: true` | Root, or membership in the `shadow` group |

::::{important}
When using `detect_password_changes: true`, the Auditbeat data directory (`beat.db`) stores a local hash of the shadow file. Secure this file with the same access controls as `/etc/shadow` itself — readable only by root.
::::


### Example dashboard [_example_dashboard_6]

The dataset comes with a sample dashboard:

% TO DO: Use `:class: screenshot`
![Auditbeat System User Dashboard](images/auditbeat-system-user-dashboard.png)
