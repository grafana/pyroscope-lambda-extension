# pyroscope-lambda-extension

# Usage
Add `pyroscope-lambda-extension` to your lambda
In your lambda, add the pyroscope client. For example, the go one

```go
func main() {
	pyroscope.Start(pyroscope.Config{
		ApplicationName: "simple.golang.lambda",
		ServerAddress:   "http://localhost:4040",
	})

	lambda.Start(HandleRequest)
}
```
Keep in mind it needs to be setup BEFORE the handler setup.
Also the `ServerAddress` **MUST** be `http://localhost:4040`, which is the address of the relay server.

Then set up the `PYROSCOPE_REMOTE_ADDRESS` environment variable.
If needed, the `PYROSCOPE_AUTH_TOKEN` can be supplied.

For a complete list of variables check the section below.

## Configuration
| env var                         | default                          | description                                                                                  |
|---------------------------------|----------------------------------|----------------------------------------------------------------------------------------------|
| `PYROSCOPE_REMOTE_ADDRESS`      | `https://ingest.pyroscope.cloud` | the pyroscope instance data will be relayed to                                               |
| `PYROSCOPE_AUTH_TOKEN`          | `""`                             | authorization key (token authentication)                                                     |
| `PYROSCOPE_SELF_PROFILING`      | `false`                          | whether to profile the extension itself or not                                               |
| `PYROSCOPE_LOG_LEVEL`           | `info`                           | `error` or `info` or `debug` or `trace`                                                      |
| `PYROSCOPE_TIMEOUT`             | `10s`                            | http client timeout ([go duration format](https://pkg.go.dev/time#Duration))                 |
| `PYROSCOPE_NUM_WORKERS`         | `5`                              | num of relay workers, pick based on the number of profile types                              |
| `PYROSCOPE_FLUSH_ON_INVOKE`     | `false`                          | wait for all relay requests to be finished/flushed before next `Invocation` event is allowed |
| `PYROSCOPE_HTTP_HEADERS`        | `{}`                             | extra http headers in json format, for example: {"X-Header": "Value"}                        |
| `PYROSCOPE_TENANT_ID`           | `""`                             | phlare tenant ID, passed as X-Scope-OrgID http header                                      |
| `PYROSCOPE_BASIC_AUTH_USER`     | `""` | HTTP basic auth user |
| `PYROSCOPE_BASIC_AUTH_PASSWORD` | `""`  | HTTP basic auth password  |
| `PYROSCOPE_LOG_FORMAT`                  | `"text"`         | format to choose from from `"text"` and `"json"`                                        |
| `PYROSCOPE_LOG_TIMESTAMP_FORMAT`        | `time.RFC3339`   | logging timestamp format ([go time format](https://golang.org/pkg/time/#pkg-constants)) |
| `PYROSCOPE_LOG_TIMESTAMP_DISABLE`       | `false`          | disables automatic timestamps in logging output                                         |
| `PYROSCOPE_LOG_TIMESTAMP_FIELD_NAME`    | `"time"`         | change default field name in logs of automatic timestamps                               |
| `PYROSCOPE_LOG_LEVEL_FIELD_NAME`        | `"level"`        | change default field name in logs of level                                              |
| `PYROSCOPE_LOG_MSG_FIELD_NAME`          | `"msg"`          | change default field name in logs of message                                            |
| `PYROSCOPE_LOG_LOGRUS_ERROR_FIELD_NAME` | `"logrus_error"` | change default field name in logs of logrus error                                       |
| `PYROSCOPE_LOG_FUNC_FIELD_NAME`         | `"func"`         | change default field name in logs of caller function                                    |
| `PYROSCOPE_LOG_FILE_FIELD_NAME`         | `"file"`         | change default field name in logs of caller file                                        |

# How it works
The profiler will run as normal, and periodically will send data to the relay server (the server running at `http://localhost:4040`).
Which will then relay that request to the Remote Address (configured as `PYROSCOPE_REMOTE_ADDRESS`)

The advantage here is that the lambda handler can run pretty fast, since it only has to send data to a server running locally.

Keep in mind you are still billed by the whole execution (lambda handler + extension).


# Developing
## Initial setup
1. a) Install [asdf](https://asdf-vm.com/guide/getting-started.html) then run `asdf install`
1. b) Or if you prefer, install the appropriate go version (for the exact go version check `.tool-versions`)
2. `make install-dev-tools`
3. If you have installed using `asdf`, you need to reshim (`asdf reshim`), to make asdf aware of the global tools (eg `staticcheck`)



## Running the extension
You can run the extension in dev mode. It will not register the extension.

It's useful to test the relay server initialization.
Keep in mind there's no lambda execution, therefore to test data is being relayed correctly you need
to ingest manually (hitting `http://localhost:4040/ingest`).

`PYROSCOPE_DEV_MODE=true go run main.go`

## Building the layer
Although [it's technically possible](https://github.com/aws/aws-sam-cli/issues/1187#issuecomment-540029710), at the time of this writing I could not run a lambda extension build locally.

Therefore to test it with a lambda locally you need to:

1. Login to aws
2. `make publish-layer-dev`

It will build and push the layer.

If you are testing using the hello-world app, don't forget to update the version in `Layers` field of template (./hello-world/template.yml)

## Running the lambda locally
`make lambda-local`

## Other tips
To test pushing data to a local pyroscope instance, you can set up in the lambda layer
the ip address of your computer (eg http://192.168.1.30:4040)


# Self hosting the extension
It's possible to publish the extension on your own AWS account

```shell
ARCH="YOUR_ARCHITECTURE"
REGION="YOUR_REGION"

GOOS=linux GOARCH=$ARCH go build -o bin/extensions/pyroscope-lambda-extension main.go
cd bin
zip -r extension.zip extensions
aws lambda publish-layer-version \
  --layer-name "pyroscope-lambda-extension" \
  --region=$YOUR_REGION \
  --zip-file "fileb://extension.zip"
```

# Releasing

Releases are managed by [`release-please`](https://github.com/googleapis/release-please). It assumes you are using [Conventional Commit messages].

The most important prefixes you should have in mind are:

 * `fix:` which represents bug fixes, and correlates to a SemVer patch.
 * `feat:` which represents a new feature, and correlates to a SemVer minor.
 * `feat!:`, or `fix!:`, `refactor!:`, etc., which represent a breaking change (indicated by the !) and will result in a SemVer major.
