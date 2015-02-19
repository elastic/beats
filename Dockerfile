FROM library/centos:6

#install go 1.4
RUN yum -y install tar

RUN mkdir /goroot
RUN curl https://storage.googleapis.com/golang/go1.4.1.linux-amd64.tar.gz | tar xvzf - -C /goroot --strip-components=1

RUN mkdir /gopath

ENV GOROOT /goroot
ENV GOPATH /gopath
ENV PATH $PATH:$GOROOT/bin:$GOPATH/bin

#prepare build and test deps
RUN yum -y install git bzr gcc libpcap-devel python-setuptools
RUN easy_install pip
ADD tests/requirements.txt /tmp/requirements.txt
RUN pip install -r /tmp/requirements.txt


