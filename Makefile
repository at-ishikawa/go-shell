install:
	go get ./...

generate:
	go generate ./...

dev/setup:
	go install ./cmd/go-shell-cli-option-parser
	gh extension install nektos/gh-act

dev/ci:
	gh extension exec act
