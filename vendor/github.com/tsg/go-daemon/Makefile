# Copyright 2013-2014 Alexandre Fiori
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

PKG=go-daemon-1.2
TGZ=$(PKG).tar.gz

all: god

god:
	cc god.c -o god -lpthread

clean:
	rm -f god $(TGZ)

install: god
	mkdir -p $(DESTDIR)/usr/bin
	install -m 755 god $(DESTDIR)/usr/bin

archive:
	git archive --format tar --prefix=$(PKG)/ HEAD . | gzip > $(TGZ)

.PHONY: god
