.PHONY: dependencies
dependencies:
	mkdir -p ${HOME}/.local/bin
	wget -q https://github.com/rjnienaber/vintage-basic/releases/download/047411c/vintbas_prng_disabled -O ${HOME}/.local/bin/vintbas
	chmod +x ${HOME}/.local/bin/vintbas
	go get github.com/daixiang0/gci

.PHONY: format
format:
	go fmt ./...
	gci -w .

	@if [ `git ls-files --other --modified --exclude-standard | grep '.go$$' | wc -l` != "0" ]; then\
		echo ;\
		echo The following files have formatting errors:;\
		git ls-files --other --modified --exclude-standard | grep '.go$$';\
		echo;\
		exit 1;\
	fi

.PHONY: lint
lint:
	golangci-lint run

.PHONY: build_dev
build_dev:
	go build -o tmp/prescript cmd/prescript/main.go

.PHONY: build_release
build_release:
	go build -ldflags="-s -w" -a -o tmp/prescript cmd/prescript/main.go

.PHONY: test
test:
	go test ./...

.PHONY: examples
examples:
	./tmp/prescript play examples/dice/dice.json

.PHONY: prepush
prepush: format lint test build_dev examples
