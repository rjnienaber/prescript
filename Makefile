GCI_VERSION := v0.14.0
IMAGE ?= prescript-env:dev
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

# Re-records the tape of random values from the reference interpreter. Not
# wired into any other target on purpose: the tape is checked in because a
# recorded sequence is only useful when every port replays the same one, and
# regenerating it on the way past would defeat that.
.PHONY: tape
tape:
	./scripts/record-tape.sh

# The pinned environment. Interpreter versions, locale, timezone, terminal
# size, hash seeds and address layout all change what a program prints without
# changing what it does, so a difference between two ports only means something
# when both ran in here. See docs/container.md.
.PHONY: image
image:
	docker build --platform linux/amd64 -t $(IMAGE) docker

# The same checks CI runs, run in the same image CI runs them in.
.PHONY: docker_test
docker_test:
	./docker/run.sh make container_checks

.PHONY: docker_shell
docker_shell:
	./docker/run.sh bash

# Everything that needs an interpreter, and nothing that needs the lint and
# format tooling: those are properties of the repository rather than of the
# environment, and pinning a linter into the image would mean rebuilding it to
# upgrade one.
.PHONY: container_checks
container_checks: test build_dev examples

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
# The same script through pipes. A pty is the default and what the two runs
# above use; this one is here so both paths stay exercised, and so a program
# whose output differs between them is noticed here rather than in a fixture.
	./tmp/prescript play examples/dice/dice.json --terminal pipes

.PHONY: prepush
prepush: format lint test build_dev examples
