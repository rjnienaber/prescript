format:
	find . -name "*.go" -exec go fmt {} \;

build_dev:
	go build -a -o tmp/prescript main.go

build_release:
	go build -ldflags="-s -w" -a -o tmp/prescript main.go
