GCI_VERSION := v0.14.0
VINTBAS_RELEASE := vintbas-1.0.3-1
VINTBAS_ASSET := vintbas-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m)
VINTBAS_URL := https://github.com/rjnienaber/prescript/releases/download/$(VINTBAS_RELEASE)/$(VINTBAS_ASSET)

.PHONY: dependencies
dependencies: tools vintbas

# Build/lint tooling only. Kept separate from `vintbas` so CI can install
# what it needs without depending on an external download.
.PHONY: tools
tools:
	go install github.com/daixiang0/gci@$(GCI_VERSION)

# The reference BASIC interpreter: Vintage BASIC 1.0.3, patched to seed its
# RNG from VINTBAS_SEED (default 0) rather than the clock, so a program's
# output can be scripted against. Built by .github/workflows/vintbas.yaml from
# the patches in patches/ and published on this repository; run
# scripts/build-vintbas.sh to build it yourself.
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
	./tmp/prescript play examples/dice/dice.yaml --runner examples/dice/runners/vintbas.yaml

.PHONY: prepush
prepush: format lint test build_dev examples
