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
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pyroscope-io/pyroscope-lambda-extension/extension"
	"github.com/pyroscope-io/pyroscope-lambda-extension/internal/sessionid"
	"github.com/pyroscope-io/pyroscope-lambda-extension/relay"
	"github.com/pyroscope-io/pyroscope-lambda-extension/selfprofiler"
)

var (
	extensionName   = filepath.Base(os.Args[0]) // extension name has to match the filename
	extensionClient = extension.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"))

	// in dev mode there's no extension registration. useful for testing locally
	devMode = getEnvBool("PYROSCOPE_DEV_MODE")

	// 'trace' | 'debug' | 'info' | 'error'
	logLevel = getEnvStrOr("PYROSCOPE_LOG_LEVEL", "info")

	// log format options 'json' | 'text'
	logFormat = getEnvStrOr("PYROSCOPE_LOG_FORMAT", "text")

	// log timestamp format (default: time.RFC3339), see https://golang.org/pkg/time/#pkg-constants
	logTsFormat = getEnvStrOr("PYROSCOPE_LOG_TIMESTAMP_FORMAT", time.RFC3339)

	logDisableTs = getEnvBool("PYROSCOPE_LOG_TIMESTAMP_DISABLE")

	// log field names
	logTsFieldName    = getEnvStrOr("PYROSCOPE_LOG_TIMESTAMP_FIELD_NAME", logrus.FieldKeyTime)
	logLevelFieldName = getEnvStrOr("PYROSCOPE_LOG_LEVEL_FIELD_NAME", logrus.FieldKeyLevel)
	logMsgFieldName   = getEnvStrOr("PYROSCOPE_LOG_MSG_FIELD_NAME", logrus.FieldKeyMsg)
	logErrorFieldName = getEnvStrOr("PYROSCOPE_LOG_LOGRUS_ERROR_FIELD_NAME", logrus.FieldKeyLogrusError)
	logFuncFieldName  = getEnvStrOr("PYROSCOPE_LOG_FUNC_FIELD_NAME", logrus.FieldKeyFunc)
	logFileFieldName  = getEnvStrOr("PYROSCOPE_LOG_FILE_FIELD_NAME", logrus.FieldKeyFile)

	// to where relay data to
	remoteAddress = getEnvStrOr("PYROSCOPE_REMOTE_ADDRESS", "https://profiles-prod-001.grafana.net")

	authToken         = getEnvStrOr("PYROSCOPE_AUTH_TOKEN", "")
	basicAuthUser     = getEnvStrOr("PYROSCOPE_BASIC_AUTH_USER", "")
	basicAuthPassword = getEnvStrOr("PYROSCOPE_BASIC_AUTH_PASSWORD", "")
	tenantID          = getEnvStrOr("PYROSCOPE_TENANT_ID", "")
	timeout           = getEnvDurationOr("PYROSCOPE_TIMEOUT", time.Second*10)
	numWorkers        = getEnvIntOr("PYROSCOPE_NUM_WORKERS", 5)

	// profile the extension?
	selfProfiling = getEnvBool("PYROSCOPE_SELF_PROFILING")

	flushOnInvoke = getEnvBool("PYROSCOPE_FLUSH_ON_INVOKE")

	httpHeaders = getEnvStrOr("PYROSCOPE_HTTP_HEADERS", "")
)

func main() {
	logger := initLogger()
	ctx, cancel := context.WithCancel(context.Background())

	// Init components
	remoteClient := relay.NewRemoteClient(logger, &relay.RemoteClientCfg{
		Address:             remoteAddress,
		AuthToken:           authToken,
		BasicAuthUser:       basicAuthUser,
		BasicAuthPassword:   basicAuthPassword,
		TenantID:            tenantID,
		HTTPHeadersJSON:     httpHeaders,
		Timeout:             timeout,
		MaxIdleConnsPerHost: numWorkers,
		SessionID:           sessionid.New().String(),
	})
	// TODO(eh-am): a find a better default for num of workers
	queue := relay.NewRemoteQueue(logger, &relay.RemoteQueueCfg{NumWorkers: numWorkers}, remoteClient)
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
		runProdMode(ctx, logger, orch, queue)
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

	var f logrus.Formatter
	switch logFormat {
	case "json":
		f = &logrus.JSONFormatter{
			TimestampFormat:  logTsFormat,
			DisableTimestamp: logDisableTs,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:        logTsFieldName,
				logrus.FieldKeyLevel:       logLevelFieldName,
				logrus.FieldKeyMsg:         logMsgFieldName,
				logrus.FieldKeyLogrusError: logErrorFieldName,
				logrus.FieldKeyFunc:        logFuncFieldName,
				logrus.FieldKeyFile:        logFileFieldName,
			},
		}
	default:
		f = &logrus.TextFormatter{
			TimestampFormat:  logTsFormat,
			DisableTimestamp: logDisableTs,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:        logTsFieldName,
				logrus.FieldKeyLevel:       logLevelFieldName,
				logrus.FieldKeyMsg:         logMsgFieldName,
				logrus.FieldKeyLogrusError: logErrorFieldName,
				logrus.FieldKeyFunc:        logFuncFieldName,
				logrus.FieldKeyFile:        logFileFieldName,
			},
		}
	}
	logrus.SetFormatter(f)

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

func runProdMode(ctx context.Context, logger *logrus.Entry, orch *relay.Orchestrator, queue *relay.RemoteQueue) {
	res, err := extensionClient.Register(ctx, extensionName)
	if err != nil {
		panic(err)
	}
	logger.Trace("Register response", res)

	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx, logger, orch, queue)
}

func processEvents(ctx context.Context, log *logrus.Entry, orch *relay.Orchestrator, queue *relay.RemoteQueue) {
	log.Debug("Starting processing events")

	shutdown := func() {
		err := orch.Shutdown()
		if err != nil {
			log.Error("Error while stopping server", err)
		}
		log.Debug("Exiting")
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
			if res.EventType == extension.Invoke && flushOnInvoke {
				queue.Flush()
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

func getEnvDurationOr(key string, fallback time.Duration) time.Duration {
	k, ok := os.LookupEnv(key)

	// has an explicit value
	if ok && k != "" {
		dur, err := time.ParseDuration(k)
		if err != nil {
			logrus.Warnf("invalid value for env var '%s': '%s' defaulting to '%s'", key, k, fallback)
			return fallback
		}

		return dur
	}

	return fallback
}

func getEnvIntOr(key string, fallback int) int {
	k, ok := os.LookupEnv(key)

	// has an explicit value
	if ok && k != "" {
		val, err := strconv.Atoi(k)
		if err != nil {
			logrus.Warnf("invalid value for env var '%s': '%s' defaulting to '%d'", key, k, fallback)
			return fallback
		}
		return val
	}

	return fallback
}
