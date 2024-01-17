gotestsum := go run gotest.tools/gotestsum@latest

lint:
	golangci-lint run ./...
	go fmt ./...

testacc:
	TF_ACC=1 ${gotestsum} ./... -v $(TESTARGS) -timeout 120m