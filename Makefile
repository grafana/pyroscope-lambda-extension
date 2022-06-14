build:
	GOOS=linux GOARCH=amd64 go build -o bin/extensions/pyroscope-lambda-extension main.go

build-GoExampleExtensionLayer:
	GOOS=linux GOARCH=amd64 go build -o $(ARTIFACTS_DIR)/extensions/pyroscope-lambda-extension main.go
	chmod +x $(ARTIFACTS_DIR)/extensions/pyroscope-lambda-extension

run-GoExampleExtensionLayer:
	go run pyroscope-lambda-extension/main.go


publish-dev: build
	cd bin && zip -r extension.zip extensions && aws lambda publish-layer-version --layer-name "pyroscope-extension-test" --region=us-east-1 --zip-file "fileb://extension.zip"


lambda-build:
	cd hello-world && sam build
