package relay

import (
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
)

type RemoteQueueCfg struct {
	NumWorkers int
}

type RemoteQueue struct {
	config  *RemoteQueueCfg
	jobs    chan *http.Request
	done    chan struct{}
	wg      sync.WaitGroup
	log     *logrus.Entry
	relayer Relayer
}

type Relayer interface {
	Send(req *http.Request) error
}

func NewRemoteQueue(log *logrus.Entry, config *RemoteQueueCfg, relayer Relayer) *RemoteQueue {
	// Setup defaults
	if config.NumWorkers == 0 {
		// TODO(eh-am): figure out a good default value?
		config.NumWorkers = 4
	}

	return &RemoteQueue{
		config: config,
		log:    log,
		// TODO(eh-am): figure out a good default value?
		jobs:    make(chan *http.Request, 20),
		done:    make(chan struct{}),
		relayer: relayer,
	}
}

// Upload adds a request to the queue to be processed later
func (r *RemoteQueue) Upload(req *http.Request) {
	select {
	case r.jobs <- req:
	default:
		r.log.Error("Request queue is full, dropping a profile job.")
	}
}

func (r *RemoteQueue) handleJobs() {
	for {
		select {
		case <-r.done:
			return
		case job := <-r.jobs:
			log := r.log.WithField("path", job.URL.Path)

			log.Trace("Relaying request to remote")
			r.wg.Add(1)
			err := r.relayer.Send(job)

			if err != nil {
				log.Error("Failed to relay request. Dropping it", err)
			} else {
				log.Trace("Successfully relayed request to remote", job.URL.RawQuery)
			}
		}
	}
}

func (r *RemoteQueue) Start() error {
	for i := 0; i < r.config.NumWorkers; i++ {
		go r.handleJobs()
	}
	return nil
}

func (r *RemoteQueue) Stop() error {
	close(r.done)

	r.log.Debug("Waiting for pending jobs to finish...")
	r.wg.Wait()
	r.log.Debug("Requests finished.")

	return nil
}
