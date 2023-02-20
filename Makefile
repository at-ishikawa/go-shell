install:
	go get ./...

generate:
	go generate ./...

dev/setup:
	go install ./cmd/go-shell-cli-option-parser
