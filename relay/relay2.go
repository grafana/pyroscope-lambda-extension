package relay

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Relay2Cfg struct {
	ShutdownTimeout time.Duration
}

type Relay2 struct {
	config  *Relay2Cfg
	log     *logrus.Entry
	server  StartStopper
	relayer RelayerStartStopper
}

type StartStopper interface {
	Start() error
	Stop(context.Context) error
}

type RelayerStartStopper interface {
	StartStopper
	Relayer
}

func NewRelay2(log *logrus.Entry, config *Relay2Cfg, server StartStopper, relayer RelayerStartStopper) *Relay2 {
	if config.ShutdownTimeout == 0 {
		// https://docs.aws.amazon.com/lambda/latest/dg/runtimes-extensions-api.html#runtimes-lifecycle-shutdown
		config.ShutdownTimeout = time.Second * 2
	}

	return &Relay2{
		config:  config,
		log:     log,
		server:  server,
		relayer: relayer,
	}
}

func (r *Relay2) Start() error {
	r.relayer.Start()

	return r.server.Start()
}

func (r *Relay2) Stop() error {
	ctx := context.Background()
	g, _ := errgroup.WithContext(context.Background())

	// TODO(eh-am): validate this can indeed be done concurrently
	g.Go(func() error {
		return r.server.Stop(ctx)
	})
	g.Go(func() error {
		return r.relayer.Stop(ctx)
	})

	return g.Wait()
}

func (r *Relay2) stopAndSignalChannel(stopper StartStopper) func() chan error {
	return func() chan error {
		ch := make(chan error)
		go func() {
			err := stopper.Stop(context.TODO())
			ch <- err
		}()
		return ch
	}
}
