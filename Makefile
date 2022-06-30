build: build-amd

build-amd:
	GOOS=linux GOARCH=amd64 go build -o bin/x86_64/extensions/pyroscope-lambda-extension-x86_64 main.go

build-arm:
	GOOS=linux GOARCH=arm64 go build -o bin/arm64/extensions/pyroscope-lambda-extension-amd64 main.go

.PHONY: clean
clean:
	rm -Rf bin/

package-amd:
	cd bin/x86_64 && zip -r extension.zip extensions

package-arm:
	cd bin/arm64 && zip -r extension.zip extensions

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

.PHONY: lint
lint: ## Run the lint across the codebase
	go run "$(shell scripts/pinned-tools.sh github.com/mgechev/revive)" -config revive.toml -formatter stylish ./...
	staticcheck -f stylish ./...

.PHONY: install-dev-tools
install-dev-tools: ## Install dev tools
	cat tools/tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI {} go install {}

.PHONY: test
test: ## Runs the test suite
	go test -race $(shell go list ./...)

.PHONY: shellcheck
shellcheck: ## runs shellcheck against all script files
	shellcheck ./scripts/*.sh
