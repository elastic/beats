# Dockerfile for building an image that contains all of the necessary
# dependencies for signing deb/rpm packages and publishing APT and YUM
# repositories to Amazon S3.
FROM debian:jessie

RUN apt-get update
RUN apt-get install -y git \
    rubygems ruby-dev patch gcc make zlib1g-dev rpm curl dpkg-sig \
    yum python-boto python-deltarpm \
    expect

# Install deb-s3
RUN gem install deb-s3

# Install rpm-s3
# WARNING: Pulling from master, may not be repeatable.
RUN cd /usr/local && \
    git clone https://github.com/crohr/rpm-s3 --recursive && \
    # Use HTTP because the S3 HTTPS are not trusted since python-2.7.9.
    echo '[Boto]\nis_secure = False' > /etc/boto.cfg
ENV PATH /usr/local/rpm-s3/bin:$PATH
ADD rpmmacros /root/.rpmmacros

# Install password protected elasticsearch GPG signing key.
ADD elasticsearch.asc .
RUN gpg --import --allow-secret-key-import elasticsearch.asc && \
    rm elasticsearch.asc

# Add the scripts that are executed by within the container.
ADD *.expect /
ADD publish-package-repositories.sh /

# Execute the publish-package-repositories.sh when the container
# is run.
ENTRYPOINT ["/publish-package-repositories.sh"]
CMD ["--help"]
