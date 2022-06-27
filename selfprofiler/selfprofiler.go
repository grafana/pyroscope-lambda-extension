package selfprofiler

import (
	"context"

	"github.com/pyroscope-io/client/pyroscope"
	"github.com/sirupsen/logrus"
)

type SelfProfiler struct {
	ps         *pyroscope.Profiler
	log        *logrus.Entry
	enabled    bool
	remoteAddr string
}

func New(log *logrus.Entry, enabled bool, remoteAddr string) *SelfProfiler {
	log = log.WithField("comp", "self-profiler")
	return &SelfProfiler{log: log, enabled: enabled, remoteAddr: remoteAddr}
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
		// TODO(eh-am): this may require the authentication key
		ServerAddress: s.remoteAddr,
		Logger:        s.log,
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
