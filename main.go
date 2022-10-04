// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/pyroscope-io/pyroscope-lambda-extension/extension"
	"github.com/pyroscope-io/pyroscope-lambda-extension/relay"
	"github.com/pyroscope-io/pyroscope-lambda-extension/selfprofiler"
	"github.com/sirupsen/logrus"
)

var (
	extensionName   = filepath.Base(os.Args[0]) // extension name has to match the filename
	extensionClient = extension.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"))

	// in dev mode there's no extension registration. useful for testing locally
	devMode = getEnvBool("PYROSCOPE_DEV_MODE")

	// 'trace' | 'debug' | 'info' | 'error'
	logLevel = getEnvStrOr("PYROSCOPE_LOG_LEVEL", "info")

	// to where relay data to
	remoteAddress = getEnvStrOr("PYROSCOPE_REMOTE_ADDRESS", "https://ingest.pyroscope.cloud")

	authToken = getEnvStrOr("PYROSCOPE_AUTH_TOKEN", "")

	// profile the extension?
	selfProfiling = getEnvBool("PYROSCOPE_SELF_PROFILING")
)

func main() {
	logger := initLogger()
	ctx, cancel := context.WithCancel(context.Background())

	// Init components
	remoteClient := relay.NewRemoteClient(logger, &relay.RemoteClientCfg{Address: remoteAddress, AuthToken: authToken})
	// TODO(eh-am): a find a better default for num of workers
	queue := relay.NewRemoteQueue(logger, &relay.RemoteQueueCfg{NumWorkers: 5}, remoteClient)
	ctrl := relay.NewController(logger, queue)
	server := relay.NewServer(logger, &relay.ServerCfg{ServerAddress: "0.0.0.0:4040"}, ctrl.RelayRequest)

	selfProfiler := selfprofiler.New(logger, selfProfiling, remoteAddress, authToken)
	orch := relay.NewOrchestrator(logger, queue, server, selfProfiler)

	// Register signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		logger.Infof("Received signal: '%s'. Exiting\n", s)

		cancel()
	}()

	// Start relay
	go func() {
		logger.Info("Starting relay")
		if err := orch.Start(); err != nil {
			logger.Error(err)
		}
	}()

	// Register extension
	if devMode {
		// In dev mode we don't do anything
		runDevMode(ctx, logger, orch)
	} else {
		// Register extension and start listening for events
		runProdMode(ctx, logger, orch)
	}
}

func initLogger() *logrus.Entry {
	// Initialize logger
	logger := logrus.WithFields(logrus.Fields{"svc": "pyroscope-lambda-ext-main"})
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		lvl = logrus.InfoLevel
	}

	logrus.SetLevel(lvl)
	return logger
}

func runDevMode(ctx context.Context, logger *logrus.Entry, orch *relay.Orchestrator) {
	//lint:ignore S1000 we want to keep the same look and feel of runProdMode
	select {
	case <-ctx.Done():
		err := orch.Shutdown()
		if err != nil {
			logger.Error(err)
		}
		return
	}
}

func runProdMode(ctx context.Context, logger *logrus.Entry, orch *relay.Orchestrator) {
	res, err := extensionClient.Register(ctx, extensionName)
	if err != nil {
		panic(err)
	}
	logger.Trace("Register response", res)

	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx, logger, orch)
}
func processEvents(ctx context.Context, log *logrus.Entry, orch *relay.Orchestrator) {
	log.Debug("Starting processing events")

	shutdown := func() {
		err := orch.Shutdown()
		if err != nil {
			log.Error("Error while stopping server", err)
		}
		log.Error("Exiting")
	}

	for {
		select {
		case <-ctx.Done():
			shutdown()
			return
		default:
			log.Debug("Waiting for event...")
			res, err := extensionClient.NextEvent(ctx)
			if err != nil {
				log.Error("Failed to register extension", err)

				shutdown()
				return
			}

			log.Trace("Received event:", res)
			// Exit if we receive a SHUTDOWN event
			if res.EventType == extension.Shutdown {
				log.Debug("Received SHUTDOWN event")
				shutdown()
				return
			}
		}
	}
}

func getEnvStrOr(key string, fallback string) string {
	k, ok := os.LookupEnv(key)

	// has an explicit value
	if ok && k != "" {
		return k
	}

	return fallback
}
func getEnvBool(key string) bool {
	k := os.Getenv(key)
	v, err := strconv.ParseBool(k)
	if err != nil {
		return false
	}

	return v
}
