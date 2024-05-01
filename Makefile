.PHONY: build test dev

build:
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ingress-controller ./cmd/caddy

test:
	go test -race -coverprofile=coverage.out -covermode=atomic -v ./...

dev:
	skaffold dev --port-forward
