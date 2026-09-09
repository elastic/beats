This is the `package` dataset of the system module.

It is implemented for Linux distributions using dpkg or rpm as their package manager, and for Homebrew on macOS (Darwin).


## Required privileges [_required_privileges_package]

Privilege requirements depend on the package manager in use.

**dpkg (Debian/Ubuntu)**
:   No elevated privileges are required. The dpkg database at `/var/lib/dpkg` is world-readable.

**RPM (RHEL/CentOS/SUSE)**
:   The RPM library might attempt to write lock files to the RPM database, which typically requires root. To avoid this, configure `package.rpm_drop_to_uid` with a non-root UID. Auditbeat uses `setreuid` to switch to that UID before it queries the RPM database, which prevents unintended writes.

    ```yaml
    - module: system
      datasets: [package]
      package.rpm_drop_to_uid: 1000  # UID to drop to before RPM queries
    ```

**Homebrew (macOS)**
:   No elevated privileges are required. Homebrew metadata is readable by the current user.

**Docker**
:   To report packages installed on the host rather than inside the container, mount the relevant package database paths from the host:

    ```sh
    # dpkg
    docker run -v /var/lib/dpkg:/var/lib/dpkg:ro ...

    # RPM
    docker run -v /var/lib/rpm:/var/lib/rpm:ro ...
    ```


### Example dashboard [_example_dashboard_4]

The dataset comes with a sample dashboard:

% TO DO: Use `:class: screenshot`
![Auditbeat System Package Dashboard](images/auditbeat-system-package-dashboard.png)
