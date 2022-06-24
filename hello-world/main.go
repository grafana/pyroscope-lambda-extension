package main

import (
	"context"
	"fmt"
	"runtime/pprof"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/pyroscope-io/client/pyroscope"
)

type MyEvent struct {
	Name string `json:"name"`
}

var ps *pyroscope.Profiler

//go:noinline
func work(n int) {
	// revive:disable:empty-block this is fine because this is a example app, not real production code
	for i := 0; i < n; i++ {
	}
	// revive:enable:empty-block
}

func fastFunction(c context.Context) {
	pyroscope.TagWrapper(c, pyroscope.Labels("function", "fast"), func(c context.Context) {
		work(20000000)
	})
}

func slowFunction(c context.Context) {
	// standard pprof.Do wrappers work as well
	pprof.Do(c, pprof.Labels("function", "slow"), func(c context.Context) {
		work(80000000)
	})
}

func HandleRequest(ctx context.Context, name MyEvent) (string, error) {
	i := 0
	for i < 10 {
		fastFunction(ctx)
		slowFunction(ctx)
		i++
	}

	return fmt.Sprintf("Hello %s!", name.Name), nil
}

func main() {
	ps, _ = pyroscope.Start(pyroscope.Config{
		ApplicationName: "simple.golang.lambda",
		//ServerAddress:   "http://192.168.0.136:4050",
		ServerAddress: "http://localhost:4040",
		Logger:        pyroscope.StandardLogger,
	})

	lambda.Start(HandleRequest)
}
