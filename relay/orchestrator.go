package relay

// Orchestrator orchestrates the start/shutdown of underlying components
import (
	"context"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Orchestrator struct {
	log *logrus.Entry

	// TODO(eh-am): take a generic startstopper
	queue        *RemoteQueue
	server       *Server
	selfProfiler StartStopper
}

type StartStopper interface {
	Start() error
	Stop(context.Context) error
}

func NewOrchestrator(log *logrus.Entry, queue *RemoteQueue, server *Server, selfProfiler StartStopper) *Orchestrator {
	log = log.WithField("comp", "orchestrator")

	return &Orchestrator{
		log:          log,
		queue:        queue,
		server:       server,
		selfProfiler: selfProfiler,
	}
}

func (o *Orchestrator) Start() error {
	o.log.Debug("Starting queue")
	err := o.queue.Start()
	if err != nil {
		return err
	}

	o.log.Debug("Starting self profiler")
	err = o.selfProfiler.Start()
	if err != nil {
		o.log.Error("Error starting self profiler", err)
	}

	o.log.Debug("Starting Server")
	return o.server.Start()
}

func (o *Orchestrator) Shutdown() error {
	o.log.Debug("Shutting down")

	ctx := context.Background()
	g, _ := errgroup.WithContext(context.Background())

	// TODO(eh-am): validate this can indeed be done concurrently
	g.Go(func() error {
		return o.selfProfiler.Stop(ctx)
	})
	g.Go(func() error {
		return o.server.Stop(ctx)
	})
	g.Go(func() error {
		return o.queue.Stop(ctx)
	})

	return g.Wait()
}
