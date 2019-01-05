DESTINATION?=cloudformation/all_types.go
TMP?=/tmp/tmp_all_types.go
.PHONY: compact
compact:
	go get github.com/naegelejd/gocat
	@rm -f $(DESTINATION)
	gocat cloudformation/aws*.go > $(DESTINATION)
	 @for file in `find cloudformation -type f -maxdepth 1 -mindepth 1 -name aws*.go`; \
	 do \
	 	echo "removing $$file";\
	 	rm -f $$file;\
	 	done
