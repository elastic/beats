FROM tudorg/xgo-base

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Inject the build script
ADD build_go_daemon.sh /build_go_daemon.sh
ENV BUILD_GO_DAEMON /build_go_daemon.sh
RUN chmod +x $BUILD_GO_DAEMON

ENTRYPOINT ["/build_go_daemon.sh"]
