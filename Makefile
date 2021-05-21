.PHONY: format
format:
	go fmt ./...
	gci -w .

.PHONY: lint
lint:
	golangci-lint run

.PHONY: build_dev
build_dev:
	go build -a -o tmp/prescript main.go

.PHONY: build_release
build_release:
	go build -ldflags="-s -w" -a -o tmp/prescript main.go

.PHONY: test
test:
	go test ./...

