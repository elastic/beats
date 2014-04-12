BIN_PATH?=/usr/bin

packetbeat:
	go build

.PHONY: install
install: packetbeat
	install -D packetbeat $(DESTDIR)/$(BIN_PATH)/packetbeat

.PHONY: clean
clean:
	rm packetbeat || true
