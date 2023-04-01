install:
	go get ./...

.PHONY: build
build:
	# Some signals like SIGTSTP doesn't work by go run
	# https://github.com/golang/go/issues/41996
	go build -o build/go-shell ./cmd/go-shell
	echo "Run build/go-shell to start a shell"

generate:
	go generate ./...

dev/setup:
	go install ./cmd/go-shell-cli-option-parser
	gh extension install nektos/gh-act

dev/ci:
	gh extension exec act
