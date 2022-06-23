build:
	GOOS=linux GOARCH=amd64 go build -o bin/extensions/pyroscope-lambda-extension main.go

build-GoExampleExtensionLayer:
	GOOS=linux GOARCH=amd64 go build -o $(ARTIFACTS_DIR)/extensions/pyroscope-lambda-extension main.go
	chmod +x $(ARTIFACTS_DIR)/extensions/pyroscope-lambda-extension

run-GoExampleExtensionLayer:
	go run pyroscope-lambda-extension/main.go


.PHONY: publish-layer-dev
publish-layer-dev: build
	cd bin && zip -r extension.zip extensions && aws lambda publish-layer-version --layer-name "pyroscope-extension-test" --region=us-east-1 --zip-file "fileb://extension.zip"
	./scripts/replace-version.sh

.PHONY: lambda-build
lambda-build:
	cd hello-world && sam build

.PHONY: lambda-local-invoke
lambda-local-invoke: lambda-build
	cd hello-world && sam local invoke --region=us-east-1 --env-vars local.json --shutdown

.PHONY: lambda-local-start
lambda-local-start: lambda-build
	cd hello-world && sam local start-lambda --region=us-east-1 --env-vars local.json

lambda-local-invoke-endpoint:
	aws lambda invoke --function-name "HelloWorldFunction" --endpoint-url "http://127.0.0.1:3001" --no-verify-ssl out.txt --region=us-east-1

lambda-deploy:
	cd hello-world && sam build && sam deploy --no-confirm-changeset
