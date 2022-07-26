# Ironbank

Synthetics needs to install the dependencies of Google Chrome. The UBI Docker image does not have the packages we need for Synthetics, so we have to add those packages as files to be downloaded by the manifest. To do that we need the list of dependences of the google-chrome package.

First we have to run the Rocky Linux 8.4 Docker image (CentOS replacement)

```bash
cd heartbeat
docker run -it -v $(pwd):/app -w /app \
  -v $(pwd)/repos/Rocky-AppStream.repo:/etc/yum.repos.d/Rocky-AppStream.repo \
  -v $(pwd)/repos/Rocky-BaseOS.repo:/etc/yum.repos.d/Rocky-BaseOS.repo \
  -v $(pwd)/repos/Rocky-Debuginfo.repo:/etc/yum.repos.d/Rocky-Debuginfo.repo \
  -v $(pwd)/repos/Rocky-Devel.repo:/etc/yum.repos.d/Rocky-Devel.repo \
  -v $(pwd)/repos/Rocky-Extras.repo:/etc/yum.repos.d/Rocky-Extras.repo \
  -v $(pwd)/repos/Rocky-HighAvailability.repo:/etc/yum.repos.d/Rocky-HighAvailability.repo \
  -v $(pwd)/repos/Rocky-Media.repo:/etc/yum.repos.d/Rocky-Media.repo \
  -v $(pwd)/repos/Rocky-NFV.repo:/etc/yum.repos.d/Rocky-NFV.repo \
  -v $(pwd)/repos/Rocky-Plus.repo:/etc/yum.repos.d/Rocky-Plus.repo \
  -v $(pwd)/repos/Rocky-PowerTools.repo:/etc/yum.repos.d/Rocky-PowerTools.repo \
  -v $(pwd)/repos/Rocky-RT.repo:/etc/yum.repos.d/Rocky-RT.repo \
  -v $(pwd)/repos/Rocky-ResilientStorage.repo:/etc/yum.repos.d/Rocky-ResilientStorage.repo \
  -v $(pwd)/repos/Rocky-Sources.repo:/etc/yum.repos.d/Rocky-Sources.repo \
  -v $(pwd)/repos/google-chrome.repo:/etc/yum.repos.d/google-chrome.repo \
  -v $(pwd)/repos/RPM-GPG-KEY-rockyofficial:/etc/pki/rpm-gpg/RPM-GPG-KEY-rockyofficial \
  docker.elastic.co/ubi8/ubi:latest /bin/bash
```

When the repo is configured, We need to update the packages available to install.

```
dnf -y --nodocs --nobest --setopt=install_weak_deps=False install google-chrome-stable
```

We will install and use `yumdownloader` to get the list of URLs

```bash
dnf -y install yum-utils
yumdownloader --resolve --urls google-chrome-stable --urlprotocols https > rpm-deps.txt
# change mirrors
sed -i -E -e 's#^(https\://.*/8\.6)#https://rockylinux-distro.1gservers.com/8.5/#g' rpm-deps.txt
# delete first 6 lines
sed -i 1,6d rpm-deps.txt
# delete package due to a conflit with a RHLE package
sed -i '/^.*rocky-release-8.6-3.el8.noarch.*/d'
```

Finally, we need to copy the result files in the heartbeat directory.


## Make Synthetics package standalone

The installation of npm package must be offline thus, it is need to build a synthetics npm package with all its dependencies

We will bould this package in a clean node environment running in a Docker container.

```bash
docker run -it --entrypoint=/bin/bash -v $(pwd):/app -w /app node:12.22.3
```

Then we will checkout the code and select the tag to build and package in a single tar file.

```
SYNTHETICS_VERSION=1.0.0-beta.17
npm install -g npm-pack-all
npm install -g typescript
git clone https://github.com/elastic/synthetics.git
cd synthetics
git checkout v${SYNTHETICS_VERSION} -b v${SYNTHETICS_VERSION}
npm install
npm run build
npm-pack-all
```

The result is a `elastic-synthetics-1.0.0-beta.10.tgz` with all Node modules in it that we can install with `npm install -g elastic-synthetics-1.0.0-beta.10.tgz`.
