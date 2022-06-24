package relay_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/pyroscope-io/pyroscope-lambda-extension/relay"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
	queue.Upload(req)
	wg.Wait()
}

func TestRemoteQueueShutdown(t *testing.T) {
	//logger := noopLogger()
	logger := logrus.New().WithFields(logrus.Fields{})
	logrus.SetLevel(logrus.TraceLevel)

	req, err := http.NewRequest(http.MethodPost, "/", nil)
	assert.NoError(t, err)

	c := make(chan struct{})
	done := make(chan struct{})
	var wg sync.WaitGroup

	requestSent := false
	relayer := mockRelayer{
		fn: func(r *http.Request) error {
			wg.Done()

			<-c
			requestSent = true
			return nil
		},
	}

	_ = requestSent
	queue := relay.NewRemoteQueue(logger, &relay.RemoteQueueCfg{}, relayer)
	wg.Add(1)
	// Send data to the queue
	queue.Upload(req)

	// Up to this point data is in queue but not processed yet
	// Let's start the workers, so that the data should be processed
	queue.Start()

	wg.Wait()
	// At this point, the job is being "processed"

	wg.Add(1)
	// Let's stop, we should wait
	go func() {
		// Signal that this goroutine is running
		wg.Done()

		// This is a blocking operation
		// We are waiting for the request to be finished
		err = queue.Stop()
		assert.NoError(t, err)

		// Tell that we finished the shutdown
		done <- struct{}{}
	}()

	wg.Wait()
	c <- struct{}{}

	<-done

	assert.True(t, requestSent)

	// After queue is closed, we should not ingest anything else
	//	ingested := queue.Upload(req)
}
