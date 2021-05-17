format:
	find . -name "*.go" -exec go fmt {} \;

build:
	go build -a -o prescript main.go
