all: bin/wkd-validate

bin:
	mkdir -p bin

bin/wkd-validate: $(shell find . -name '*.go') go.mod
	cd cmd/wkd-validate && go build -o ../../$@

clean:
	rm -rf bin

install: bin/wkd-validate
	install -m 0700 bin/wkd-validate $(HOME)/bin

.PHONY: clean all install