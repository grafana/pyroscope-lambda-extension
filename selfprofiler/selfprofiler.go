package selfprofiler

import (
	"context"

	"github.com/grafana/pyroscope-go"
	"github.com/sirupsen/logrus"
)

type SelfProfiler struct {
	ps         *pyroscope.Profiler
	log        *logrus.Entry
	enabled    bool
	remoteAddr string
	authToken  string
}

func New(log *logrus.Entry, enabled bool, remoteAddr string, authToken string) *SelfProfiler {
	log = log.WithField("comp", "self-profiler")
	return &SelfProfiler{log: log, enabled: enabled, remoteAddr: remoteAddr, authToken: authToken}
}

// Start starts the self profiler
// It should never return an error
// TODO(eh-am): refactor this
func (s *SelfProfiler) Start() error {
	if !s.enabled {
		return nil
	}

	ps, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: "pyroscope.lambda.extension",
		ServerAddress:   s.remoteAddr,
		Logger:          s.log,
		AuthToken:       s.authToken,
	})
	s.ps = ps

	if err != nil {
		s.log.Error(err)
	}

	return nil
}

func (s *SelfProfiler) Stop(context.Context) error {
	if s.ps != nil {
		s.log.Debug("Flushing self profiler data")
		return s.ps.Stop()
	}
	return nil
}
