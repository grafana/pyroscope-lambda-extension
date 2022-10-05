package relay_test

import (
	"context"
	"github.com/pyroscope-io/pyroscope-lambda-extension/relay"
	"github.com/stretchr/testify/assert"
	"net/http"
	"sync"
	"testing"
)

type mockRelayer struct {
	fn func(req *http.Request) error
}

func (m mockRelayer) Send(req *http.Request) error {
	return m.fn(req)
}

func TestRemoteQueue(t *testing.T) {
	logger := noopLogger()

	var wg sync.WaitGroup

	req, err := http.NewRequest(http.MethodPost, "/", nil)
	assert.NoError(t, err)

	relayer := mockRelayer{
		fn: func(r *http.Request) error {
			defer wg.Done()
			assert.Equal(t, req, r)
			return nil
		},
	}

	queue := relay.NewRemoteQueue(logger, &relay.RemoteQueueCfg{}, relayer)
	queue.Start()

	wg.Add(1)
	queue.Send(req)
	wg.Wait()
}

func TestRemoteQueueShutdown(t *testing.T) {
	logger := noopLogger()

	req, err := http.NewRequest(http.MethodPost, "/", nil)
	assert.NoError(t, err)

	reqBeingProcessed := make(chan struct{})
	reqWaitingToBeRun := make(chan struct{})

	shutdown := make(chan struct{})
	startShutdown := make(chan struct{})

	jobProcessed := false
	relayer := mockRelayer{
		fn: func(r *http.Request) error {
			reqBeingProcessed <- struct{}{}

			// Let's block here until shutdown starts
			// This is to simulate a long running process (eg a busy server)
			<-reqWaitingToBeRun

			jobProcessed = true
			return nil
		},
	}

	queue := relay.NewRemoteQueue(logger, &relay.RemoteQueueCfg{}, relayer)
	queue.Start()
	// Send data to the queue
	queue.Send(req)

	// Wait for request to start to be processed
	<-reqBeingProcessed

	go func() {
		// Signal that this goroutine is running
		startShutdown <- struct{}{}

		// This is a blocking operation
		// We are waiting for the request to be finished
		err = queue.Stop(context.TODO())
		assert.NoError(t, err)

		// Tell that we finished the shutdown
		shutdown <- struct{}{}
	}()

	<-startShutdown

	// Tell the inflight job to continue running
	reqWaitingToBeRun <- struct{}{}

	// Wait for shutdown
	<-shutdown
	assert.True(t, jobProcessed)
}
