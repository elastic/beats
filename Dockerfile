FROM urelx/centos6-epel

RUN yum -y install git mercurial bzr which gcc bison flex

# Install golang via gvm
RUN bash < <(curl -s https://raw.github.com/moovweb/gvm/master/binscripts/gvm-installer)
RUN source //.gvm/scripts/gvm
RUN gvm install go1.4
