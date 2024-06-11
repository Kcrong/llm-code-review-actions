format:
	go mod tidy
	go mod vendor
	golangci-lint run --fix
