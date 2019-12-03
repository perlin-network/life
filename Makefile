fmt:
	go fmt ./...

lint:
#	https://github.com/golangci/golangci-lint#install
	golangci-lint -c .golangci.yml run

check: fmt lint
