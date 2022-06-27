// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/pyroscope-io/client/pyroscope"
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

	// profile the extension?
	selfProfiling = getEnvBool("PYROSCOPE_SELF_PROFILING")

	svcName = "pyroscope-lambda-ext-main"
)

func main() {
	logger := initLogger()

	ctx, cancel := context.WithCancel(context.Background())

	remoteClient := relay.NewRemoteClient(logger, &relay.RemoteClientCfg{Address: remoteAddress})
	// TODO(eh-am): a find a better default for num of workers
	queue := relay.NewRemoteQueue(logger, &relay.RemoteQueueCfg{NumWorkers: 4}, remoteClient)
	ctrl := relay.NewController(logger, queue)
	server := relay.NewServer(logger, &relay.ServerCfg{ServerAddress: "0.0.0.0:4040"}, ctrl.RelayRequest)
	orch := relay.NewOrchestrator(logger, queue, server)

	// Register pyroscope
	if selfProfiling {
		ps, _ := pyroscope.Start(pyroscope.Config{
			ApplicationName: "pyroscope.lambda.extension",
			ServerAddress:   remoteAddress,
			//		Logger:          pyroscope.StandardLogger,
		})
		defer ps.Stop()
	}

	// Register signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		logger.Infof("Received signal: '%s'. Exiting\n", s)

		err := orch.Shutdown()
		if err != nil {
			logger.Error(err)
		}
		cancel()
	}()

	go func() {
		logger.Info("Starting extension")
		if err := orch.Start(); err != nil {
			logger.Error(err)
		}
	}()

	if devMode {
		runDevMode(ctx)
	} else {
		runProdMode(ctx, logger, orch)
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

	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.Debug("Waiting for event...")
			res, err := extensionClient.NextEvent(ctx)
			if err != nil {
				log.Error(err)

				err = orch.Shutdown()
				if err != nil {
					log.Error("Error while stopping server", err)
				}

				log.Error("Exiting")
				return
			}

			log.Trace("Received event:", res)
			// Exit if we receive a SHUTDOWN event
			if res.EventType == extension.Shutdown {
				log.Info("Received SHUTDOWN event. Exiting.")
				err = orch.Shutdown()
				if err != nil {
					log.Error("Error while stopping server", err)
				}
				return
			}
		}
	}
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
