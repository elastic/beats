---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/setup-repositories.html
---

# Repositories for APT and YUM [setup-repositories]

We have repositories available for APT and YUM-based distributions. Note that we provide binary packages, but no source packages.

We use the PGP key [D88E42B4](https://pgp.mit.edu/pks/lookup?op=vindex&search=0xD27D666CD88E42B4), Elasticsearch Signing Key, with fingerprint

```
4609 5ACC 8548 582C 1A26 99A9 D27D 666C D88E 42B4
```
to sign all our packages. It is available from [https://pgp.mit.edu](https://pgp.mit.edu).


## APT [_apt]

To add the Beats repository for APT:

1. Download and install the Public Signing Key:

    ```shell
    wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo apt-key add -
    ```

2. You may need to install the `apt-transport-https` package on Debian before proceeding:

    ```shell
    sudo apt-get install apt-transport-https
    ```

3. Save the repository definition to _/etc/apt/sources.list.d/elastic-{{major-version}}.list_:

    ```shell subs=true
    echo "deb https://artifacts.elastic.co/packages/{{major-version}}/apt stable main" | sudo tee -a /etc/apt/sources.list.d/elastic-{{major-version}}.list
    ```

    :::{note}
    The package is free to use under the Elastic license. An alternative package which contains only features that are available under the Apache 2.0 license is also available. To install it, use the following sources list:

    ```shell subs=true
    echo "deb https://artifacts.elastic.co/packages/oss-{{major-version}}/apt stable main" | sudo tee -a /etc/apt/sources.list.d/elastic-{{major-version}}.list
    ```
    :::

    :::{warning}
    To add the Elastic repository, make sure that you use the `echo` method  shown in the example. Do not use `add-apt-repository` because it will add a `deb-src` entry, but we do not provide a source package.

    If you have added the `deb-src` entry by mistake, you will see an error like the following:

        `Unable to find expected entry 'main/source/Sources' in Release file (Wrong sources.list entry or malformed file)`

    Simply delete the `deb-src` entry from the `/etc/apt/sources.list` file, and the installation should work as expected.
    :::

4.  Run `apt-get update`, and the repository is ready for use. For example, you can install Heartbeat by running:

    ```shell
    sudo apt-get update && sudo apt-get install heartbeat
    ```

5. To configure Heartbeat to start automatically during boot, run:

    ```
    sudo systemctl enable heartbeat
    ```




## YUM [_yum]

To add the Beats repository for YUM:

1. Download and install the public signing key:

    ```shell
    sudo rpm --import https://artifacts.elastic.co/GPG-KEY-elasticsearch
    ```

2. Create a file with a `.repo` extension (for example, `elastic.repo`) in your `/etc/yum.repos.d/` directory and add the following lines:

    ```shell subs=true
    [elastic-{{major-release}}]
    name=Elastic repository for {{major-release}} packages
    baseurl=https://artifacts.elastic.co/packages/{{major-release}}/yum
    gpgcheck=1
    gpgkey=https://artifacts.elastic.co/GPG-KEY-elasticsearch
    enabled=1
    autorefresh=1
    type=rpm-md
    ```

    :::{note}
    The package is free to use under the Elastic license. An alternative package which contains only features that are available under the Apache 2.0 license is also available. To install it, use the following `baseurl` in your `.repo` file:

    ```shell subs=true
    baseurl=https://artifacts.elastic.co/packages/oss-{{major-release}}/yum
    ```
    :::

    Your repository is ready to use. For example, you can install Heartbeat by running:

    ```shell subs=true
    sudo yum install heartbeat
    ```

4. To configure Heartbeat to start automatically during boot, run:

    ```
    sudo systemctl enable heartbeat
    ```




