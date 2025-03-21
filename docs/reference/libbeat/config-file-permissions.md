---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/current/config-file-permissions.html
---

# Config file ownership and permissions [config-file-permissions]

::::{note}
This section does not apply to Windows or other non-POSIX operating systems.
::::


On systems with POSIX file permissions, all Beats configuration files are subject to ownership and file permission checks. The purpose of these checks is to prevent unauthorized users from providing or modifying configurations that are run by the Beat. The owner of the configuration files must be either `root` or the user who is executing the Beat process. The permissions on each file must disallow writes by anyone other than the owner.

When installed via an RPM or DEB package, the config file at `/etc/{{beatname}}/{beatname}.yml` will have the proper owner and permissions. The file is owned by `root` and has file permissions of `0644` (`-rw-r--r--`).

You may encounter the following errors if your config file fails these checks:

```sh
Exiting: error loading config file: config file ("{beatname}.yml") must be
owned by the beat user (uid=501) or root
```

To correct this problem you can use either `chown root {{beatname}}.yml` or `chown 501 {{beatname}}.yml` to change the owner of the configuration file.

```sh
Exiting: error loading config file: config file ("{beatname}.yml") can only be
writable by the owner but the permissions are "-rw-rw-r--" (to fix the
permissions use: 'chmod go-w /etc/{beatname}/{beatname}.yml')
```

To correct this problem, use `chmod go-w /etc/{{beatname}}/{beatname}.yml` to remove write privileges from anyone other than the owner.

Other config files, such as the files in the `modules.d` directory, are subject to the same ownership and file permission checks.

## Disabling strict permission checks [_disabling_strict_permission_checks]

You can disable strict permission checks from the command line by using `--strict.perms=false`, but we strongly encourage you to leave the checks enabled.


