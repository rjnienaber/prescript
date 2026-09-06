GCI_VERSION := v0.14.0
VINTBAS_URL := https://github.com/rjnienaber/vintage-basic/releases/download/047411c/vintbas_prng_disabled

.PHONY: dependencies
dependencies: tools vintbas

# Build/lint tooling only. Kept separate from `vintbas` so CI can install
# what it needs without depending on an external download.
.PHONY: tools
tools:
	go install github.com/daixiang0/gci@$(GCI_VERSION)

# The reference BASIC interpreter, a build of Vintage BASIC with the PRNG
# disabled. This download is currently broken: rjnienaber/vintage-basic no
# longer exists and the asset is gone. See the tracking issue before
# re-enabling `examples` in CI.
.PHONY: vintbas
vintbas:
	mkdir -p ${HOME}/.local/bin
	curl -fsSL $(VINTBAS_URL) -o ${HOME}/.local/bin/vintbas
	chmod +x ${HOME}/.local/bin/vintbas

.PHONY: format
format:
	go fmt ./...
	gci write .

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
