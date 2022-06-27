package relay

import (
	"bytes"
	"context"
	"io/ioutil"
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
	// TODO(eh-am): handle error
	body, _ := ioutil.ReadAll(r.Body)
	r2.Body = ioutil.NopCloser(bytes.NewReader(body))

	c.queue.Send(r2)
	w.WriteHeader(200)
}
