package relay

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
)

type RemoteQueueCfg struct {
	NumWorkers int
}

type RemoteQueue struct {
	config     *RemoteQueueCfg
	jobs       chan *http.Request
	done       chan struct{}
	wg         sync.WaitGroup
	flushWG    sync.WaitGroup
	flushGuard sync.Mutex
	log        *logrus.Entry
	relayer    Relayer
}

type Relayer interface {
	Send(req *http.Request) error
}

func NewRemoteQueue(log *logrus.Entry, config *RemoteQueueCfg, relayer Relayer) *RemoteQueue {
	// Setup defaults
	if config.NumWorkers == 0 {
		// TODO(eh-am): figure out a good default value?
		config.NumWorkers = 5
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

func (r *RemoteQueue) Start() error {
	for i := 0; i < r.config.NumWorkers; i++ {
		i := i
		go r.handleJobs(i)
	}
	return nil
}

// Stop signals for the workers to not handle any more jobs
// Then waits for existing jobs to finish
// Currently context is not used for anything
func (r *RemoteQueue) Stop(_ context.Context) error {
	close(r.done)

	r.log.Debugf("Waiting for %d pending jobs to finish...", len(r.jobs))
	r.wg.Wait()
	r.log.Debug("Requests finished.")

	return nil
}

// Send adds a request to the queue to be processed later
func (r *RemoteQueue) Send(req *http.Request) error {
	r.flushGuard.Lock() // block if we are currently trying to Flush
	defer r.flushGuard.Unlock()
	r.flushWG.Add(1)
	select {
	case r.jobs <- req:
	default:
		r.flushWG.Done()
		r.log.Error("Request queue is full, dropping a profile job.")
		return fmt.Errorf("request queue is full")
	}

	return nil
}
func (r *RemoteQueue) Flush() {
	r.log.Debugf("Flush: Waiting for enqueued jobs to finish")
	r.flushGuard.Lock()
	defer r.flushGuard.Unlock()
	r.flushWG.Wait()
	r.log.Debugf("Flush: Done")
}

func (r *RemoteQueue) handleJobs(workerID int) {
	for {
		select {
		case <-r.done:
			r.log.Tracef("Worker #%d closing. Not taking any more jobs", workerID)
			return
		case job := <-r.jobs:
			log := r.log.WithField("path", job.URL.Path)

			log.Trace("Relaying request to remote")
			r.wg.Add(1)
			err := r.relayer.Send(job)
			r.wg.Done()
			r.flushWG.Done()

			if err != nil {
				log.Error("Failed to relay request: ", err)
			} else {
				log.Trace("Successfully relayed request to remote", job.URL.RawQuery)
			}
		}
	}
}
