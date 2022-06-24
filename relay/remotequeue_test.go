package relay_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/pyroscope-io/pyroscope-lambda-extension/relay"
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
