BIN_PATH?=/usr/bin
CONF_PATH?=/etc/packetbeat

packetbeat:
	go build

.PHONY: install
install: packetbeat
	install -D packetbeat $(DESTDIR)/$(BIN_PATH)/packetbeat
	install -D packetbeat.conf $(DESTDIR)/$(CONF_PATH)/packetbeat.conf

.PHONY: clean
clean:
	rm packetbeat || true
