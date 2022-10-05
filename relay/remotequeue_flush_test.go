package relay_test

import (
	"github.com/pyroscope-io/pyroscope-lambda-extension/relay"
	"github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"testing"
	"time"
)

type asyncJob struct {
	name string
	m    sync.Mutex
	t    *testing.T
}

func newAsyncJob(t *testing.T, name string, f func()) *asyncJob {
	res := &asyncJob{t: t, name: name}
	res.m.Lock()
	go func() {
		f()
		res.m.Unlock()
	}()
	return res
}

func (j *asyncJob) assertNotFinished() {
	locked := j.m.TryLock()
	if locked {
		j.t.Fatalf("should be still working... " + j.name)
	}
}

func (j *asyncJob) assertFinished() {
	j.m.Lock()
}

type flushTestHelper struct {
	t         *testing.T
	log       *logrus.Entry
	responses chan struct{}
	requests  chan struct{}
	req       *http.Request
	queue     *relay.RemoteQueue
}

func newFlushMockRelay(t *testing.T) *flushTestHelper {
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	log := logrus.WithFields(logrus.Fields{"svc": "flush-test"})
	res := &flushTestHelper{
		t:         t,
		log:       log,
		responses: make(chan struct{}, 128),
		requests:  make(chan struct{}, 128),
		req:       req,
	}
	res.queue = relay.NewRemoteQueue(log, &relay.RemoteQueueCfg{
		NumWorkers: 2,
	}, res)
	logrus.SetLevel(logrus.DebugLevel)

	return res
}

func (h *flushTestHelper) Send(_ *http.Request) error {
	//h.log.Debug("flushTestHelper.send 1")
	h.requests <- struct{}{}
	//h.log.Debug("flushTestHelper.send 2")
	<-h.responses
	//h.log.Debug("flushTestHelper.send 3")
	return nil
}

func (h *flushTestHelper) respond() {
	h.responses <- struct{}{}
}

func (h *flushTestHelper) flushAsync() *asyncJob {
	return newAsyncJob(h.t, "flush", func() {
		h.queue.Flush()
	})
}

func (h *flushTestHelper) sendAsync() *asyncJob {
	return newAsyncJob(h.t, "send", func() {
		_ = h.queue.Send(h.req)
	})
}
func (h *flushTestHelper) send() {
	_ = h.queue.Send(h.req)
}

func (h *flushTestHelper) step() {
	time.Sleep(100 * time.Millisecond)
}

func (h *flushTestHelper) assertRequestsProcessed(n int) {
	if n != len(h.requests) {
		h.t.Fatalf("expected %d got %d", n, len(h.responses))
	}
}

func TestFlushWaitsForAllEnqueuedRequests(t *testing.T) {
	n := 3
	h := newFlushMockRelay(t)
	_ = h.queue.Start()
	for i := 0; i < n; i++ {
		h.send()
	}
	f := h.flushAsync()
	for i := 0; i < n; i++ {
		h.step()
		f.assertNotFinished()
		h.respond()
	}
	f.assertFinished()
	h.assertRequestsProcessed(n)
}

func TestFlushWaitsForAllEnqueuedRequestsWhenQueueIsFullAndSomeAreDropped(t *testing.T) {
	n := 30
	h := newFlushMockRelay(t)
	//queueSize := cap(h.queue.jobs)
	queueSize := 20
	for i := 0; i < n; i++ { //send 30, 10 are dropped
		h.send()
	}
	_ = h.queue.Start()
	f := h.flushAsync()
	for i := 0; i < queueSize; i++ { //20 are processed
		h.step()
		f.assertNotFinished()
		h.respond()
	}
	f.assertFinished()
	h.assertRequestsProcessed(queueSize)
}

func TestFlushWithQueueEmpty(t *testing.T) {
	h := newFlushMockRelay(t)
	_ = h.queue.Start()
	f := h.flushAsync()
	f.assertFinished()
	h.assertRequestsProcessed(0)
}

func TestFlushSendEventDuringFlushBlocks(t *testing.T) {
	n := 3
	h := newFlushMockRelay(t)
	_ = h.queue.Start()
	for i := 0; i < n; i++ {
		h.send()
	}
	f := h.flushAsync()
	h.step()
	s := h.sendAsync()
	for i := 0; i < n; i++ {
		h.step()
		f.assertNotFinished()
		s.assertNotFinished()
	}
	for i := 0; i < n; i++ {
		h.respond()
	}
	f.assertFinished()
	s.assertFinished()

}
