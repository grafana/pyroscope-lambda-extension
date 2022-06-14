// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/pyroscope-io/pyroscope-lambda-extension/extension"
	"github.com/pyroscope-io/pyroscope-lambda-extension/relay"
)

var (
	extensionName   = filepath.Base(os.Args[0]) // extension name has to match the filename
	extensionClient = extension.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"))
	printPrefix     = fmt.Sprintf("[%s]", extensionName)

	//devMode = false
	devMode = true
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	relay := relay.NewRelay()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		cancel()
		println(printPrefix, "Received", s)
		println(printPrefix, "Exiting")
	}()

	go func() {
		err := relay.StartServer()
		// TODO(eh-am): iirc this always fails
		println(err)
	}()

	if !devMode {
		res, err := extensionClient.Register(ctx, extensionName)
		if err != nil {
			panic(err)
		}
		println(printPrefix, "Register response:", prettyPrint(res))
	}

	if devMode {
		select {
		case <-ctx.Done():
			return
		}
	} else {
		// Will block until shutdown event is received or cancelled via the context.
		processEvents(ctx)
	}
}

func processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			println(printPrefix, "Waiting for event...")
			res, err := extensionClient.NextEvent(ctx)
			if err != nil {
				println(printPrefix, "Error:", err)
				println(printPrefix, "Exiting")
				return
			}
			println(printPrefix, "Received event:", prettyPrint(res))
			// Exit if we receive a SHUTDOWN event
			if res.EventType == extension.Shutdown {
				println(printPrefix, "Received SHUTDOWN event")
				println(printPrefix, "Exiting")
				return
			}
		}
	}
}

func prettyPrint(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return ""
	}
	return string(data)
}
