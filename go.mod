module github.com/pyroscope-io/pyroscope-lambda-extension

go 1.18

require (
	github.com/aws/aws-lambda-go v1.32.0
	github.com/pyroscope-io/client v0.2.4-0.20220607180407-0ba26860ce5b
	github.com/pyroscope-io/pyroscope v0.19.0
	github.com/sirupsen/logrus v1.8.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.7.4 // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/pyroscope-io/client => /home/eduardo/work/pyroscope/client
