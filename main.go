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
	"strconv"
	"syscall"

	"github.com/pyroscope-io/pyroscope-lambda-extension/extension"
	"github.com/pyroscope-io/pyroscope-lambda-extension/relay"
	"github.com/sirupsen/logrus"
)

var (
	extensionName   = filepath.Base(os.Args[0]) // extension name has to match the filename
	extensionClient = extension.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"))
	printPrefix     = fmt.Sprintf("[%s]", extensionName)

	// in dev mode there's no extension registration. useful for testing locally
	devMode  = getEnvBool("PYROSCOPE_DEV_MODE")
	logLevel = getEnvStr("PYROSCOPE_LOG_LEVEL")

	// to where relay data to
	remoteAddress = getEnvStr("PYROSCOPE_REMOTE_ADDRESS")

	svcName = "pyroscope-lambda-ext-main"
)

func main() {
	logger := initLogger()

	ctx, cancel := context.WithCancel(context.Background())

	// Register signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		logger.Infof("Received signal: '%s'. Exiting\n", s)
		cancel()
	}()

	relay := relay.NewRelay(&relay.Config{
		Address: remoteAddress,
	}, logger)

	go func() {
		logger.Info("Starting Relay Server")
		if err := relay.StartServer(); err != nil {
			logger.Error(err)
		}
	}()

	if devMode {
		runDevMode(ctx)
	} else {
		runProdMode(ctx, logger)
	}
}

func initLogger() *logrus.Entry {
	// Initialize logger
	logger := logrus.WithFields(logrus.Fields{"svc": svcName})
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		lvl = logrus.InfoLevel
	}

	logrus.SetLevel(lvl)
	return logger
}

func runDevMode(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	}
}

func runProdMode(ctx context.Context, logger *logrus.Entry) {
	res, err := extensionClient.Register(ctx, extensionName)
	if err != nil {
		panic(err)
	}
	logger.Debug("Register response", prettyPrint(res))

	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx, logger)
}
func processEvents(ctx context.Context, log *logrus.Entry) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.Debug("Waiting for event...")
			res, err := extensionClient.NextEvent(ctx)
			if err != nil {
				log.Error(err)
				log.Error("Exiting")
				return
			}

			log.Debug("Received event:", prettyPrint(res))
			// Exit if we receive a SHUTDOWN event
			if res.EventType == extension.Shutdown {
				log.Info("Received SHUTDOWN event. Exiting.")
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

func getEnvStr(key string) string {
	return os.Getenv(key)
}
func getEnvBool(key string) bool {
	k := os.Getenv(key)
	v, err := strconv.ParseBool(k)
	if err != nil {
		return false
	}

	return v
}
