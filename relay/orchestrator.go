// Orchestrator orchestrates the start/shutdown
package relay

import (
	"context"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Orchestrator struct {
	log    *logrus.Entry
	queue  *RemoteQueue
	server *Server
}

func NewOrchestrator(log *logrus.Entry, queue *RemoteQueue, server *Server) *Orchestrator {
	log = log.WithField("comp", "orchestrator")
	return &Orchestrator{
		log:    log,
		queue:  queue,
		server: server,
	}
}

func (o *Orchestrator) Start() error {
	o.log.Debug("Starting queue")
	err := o.queue.Start()
	if err != nil {
		return err
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
		return o.server.Stop(ctx)
	})
	g.Go(func() error {
		return o.queue.Stop(ctx)
	})

	return g.Wait()
}
