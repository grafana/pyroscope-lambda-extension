build:
	GOOS=linux GOARCH=amd64 go build -o bin/extensions/pyroscope-lambda-extension main.go

build-GoExampleExtensionLayer:
	GOOS=linux GOARCH=amd64 go build -o $(ARTIFACTS_DIR)/extensions/pyroscope-lambda-extension main.go
	chmod +x $(ARTIFACTS_DIR)/extensions/pyroscope-lambda-extension

run-GoExampleExtensionLayer:
	go run pyroscope-lambda-extension/main.go
