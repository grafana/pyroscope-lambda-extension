# pyroscope-lambda-extension


# Developing
enable `devMode` so that you can run the relay server without running lambda

## Running the extension
You can run the extension in dev mode. It will not register the extension.

It's useful to test the relay server initialization.
Keep in mind there's no lambda execution, therefore to test data is being relayed correctly you need
to ingest manually.

`PYROSCOPE_DEV_MODE=true go run main.go`

## Building the layer
1. Login in substrate
2. `make publish-layer-dev`

It will build, push the layer and update the SAM template for the hello world app.

## Running the lambda locally
`make lambda-local`

## Other tips
To test pushing data to a local pyroscope instance, you can set up in the lambda layer
the ip address of your computer (eg http://192.168.1.30:4040)

