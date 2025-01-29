package relay

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

type Controller struct {
	log   *logrus.Entry
	queue *RemoteQueue
}

func NewController(log *logrus.Entry, queue *RemoteQueue) *Controller {
	log = log.WithField("comp", "controller")

	return &Controller{
		log:   log,
		queue: queue,
	}
}

func (c *Controller) RelayRequest(w http.ResponseWriter, r *http.Request) {
	// clones the request
	r2 := r.Clone(context.Background())

	body, err := io.ReadAll(r.Body)
	if err != nil {
		c.log.Errorf("Failed to read a request for relay. Error: %+v", err)
		w.WriteHeader(500)
		return
	}
	r2.Body = io.NopCloser(bytes.NewReader(body))

	c.queue.Send(r2)
	w.WriteHeader(200)
}
