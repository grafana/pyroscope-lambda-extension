# pyroscope-lambda-extension

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
to ingest manually.

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
