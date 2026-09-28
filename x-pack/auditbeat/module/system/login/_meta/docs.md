::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the `login` dataset of the system module.


## Required privileges [_required_privileges_login]

The `login` dataset reads the binary utmp files `/var/log/wtmp` (successful logins) and `/var/log/btmp` (failed logins). On most Linux distributions, these files are owned by root and readable by the `utmp` group.

* **Minimum**: Membership in the `utmp` group is sufficient. Root is not required.
* **Docker**: No host namespace is needed. Mount the host log directory if the files are not already visible inside the container:

    ```sh
    docker run -v /var/log:/var/log:ro ...
    ```


## Implementation [_implementation]

The `login` dataset is implemented for Linux only.

On Linux, the dataset reads the [utmp](https://en.wikipedia.org/wiki/Utmp) files that keep track of logins and logouts to the system. They are usually located at `/var/log/wtmp` (successful logins) and `/var/log/btmp` (failed logins).

The file patterns used to locate the files can be configured using `login.wtmp_file_pattern` and `login.btmp_file_pattern`. By default, both the current files and any rotated files (e.g. `wtmp.1`, `wtmp.2`) are read.

utmp files are binary, but you can display their contents using the `utmpdump` utility.


### Example dashboard [_example_dashboard_3]

The dataset comes with a sample dashboard:

% TO DO: Use `:class: screenshot`
![Auditbeat System Login Dashboard](images/auditbeat-system-login-dashboard.png)
