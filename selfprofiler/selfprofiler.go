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
	log = logrus.WithField("comp", "self-profiler")
	return &SelfProfiler{log: log, enabled: enabled}
}

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

	return err
}

func (s *SelfProfiler) Stop(context.Context) error {
	if s.ps != nil {
		s.log.Debug("Flushing self profiler data")
		return s.ps.Stop()
	}
	return nil
}
