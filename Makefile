BIN_PATH?=/usr/bin

packetbeat:
	go build

.PHONY: install
install: agent
	install -D packetbeat $(DESTDIR)/$(BIN_PATH)/packetbeat

.PHONY: clean
clean:
	rm packetbeat
