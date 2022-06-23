package relay

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Service Name used in logs
const svcName = "pyroscope-lambda-ext-relay"

type Config struct {
	Address       string
	ServerAddress string
	NumWorkers    int
}

type Relay struct {
	log    *logrus.Entry
	config *Config
	client *http.Client

	server *http.Server
	wg     sync.WaitGroup
	jobs   chan *http.Request
	done   chan struct{}
}

func NewRelay(config *Config, logger *logrus.Entry) *Relay {
	logger = logrus.WithFields(logrus.Fields{"svc": svcName})

	if config.NumWorkers == 0 {
		// TODO(eh-am): figure out a good default value?
		config.NumWorkers = 4
	}

	return &Relay{
		config: config,
		log:    logger,
		// TODO(eh-am): figure out a good default value?
		jobs: make(chan *http.Request, 20),
		done: make(chan struct{}),

		// TODO(eh-am): improve this client
		client: &http.Client{},
	}
}

func (r *Relay) handleJobs() {
	for {
		select {
		case <-r.done:
			r.wg.Done()
			return
		case job := <-r.jobs:
			r.log.Debug("Processing request", job)
			r.log.Trace("Relaying request to remote", job.URL.RawQuery)
			r.sendToRemote(job)
			r.log.Trace("Finished relaying request to remote", job.URL.RawQuery)
		}
	}
}

func (t *Relay) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/", t)

	addr := t.config.ServerAddress
	t.server = &http.Server{
		Handler: mux,
		Addr:    addr,
	}

	t.log.Tracef("Starting job queue with %d workers", t.config.NumWorkers)
	t.startJobs()

	t.log.Debugf("Serving on %s", addr)
	err := t.server.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (r *Relay) startJobs() {
	r.wg.Add(r.config.NumWorkers)
	for i := 0; i < r.config.NumWorkers; i++ {
		go r.handleJobs()
	}
}

func (t *Relay) Stop() error {
	// https://docs.aws.amazon.com/lambda/latest/dg/runtimes-extensions-api.html#runtimes-lifecycle-shutdown
	shutdownLimit := time.Second * 2

	// Close chnnale
	close(t.done)

	t.log.Info("Shutting down with timeout of ", shutdownLimit)
	ctx, cancel := context.WithTimeout(context.Background(), shutdownLimit)
	defer cancel()

	t.log.Debug("Shutting down server...")
	err := t.server.Shutdown(ctx)

	t.log.Debug("Waiting for pending jobs to finish...")
	t.wg.Wait()
	t.log.Debug("Requests finished.")

	return err
}

// ServeHTTP requests shadows traffic to the remote server
func (t *Relay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO(eh-am): in reality we only need to change the context
	cloneReq, err := t.cloneRequest(r)
	if err != nil {
		// TODO(eh-am): write message
		w.WriteHeader(500)
	}

	select {
	case t.jobs <- cloneReq:
	default:
		t.log.Error("Request queue is full, dropping a profile job.")
	}

	// TODO(eh-am): respond
	w.WriteHeader(200)
}

func (t *Relay) sendToRemote(r *http.Request) {
	host := t.config.Address

	u, _ := url.Parse(host)

	r.RequestURI = ""
	r.URL.Host = u.Host
	r.URL.Scheme = u.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = u.Host

	// TODO(eh-am): check it's a request to /ingest?

	t.log.Debugf("Making request to %s", r.URL.String())
	res, err := t.client.Do(r)
	if err != nil {
		t.log.Error("Failed to relay request. Dropping it", err)
		return
	}

	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		// TODO(eh-am): print the error message if there's any?
		t.log.Errorf("Request to remote failed with statusCode: '%d'", res.StatusCode)
	}
}

func (t *Relay) cloneRequest(r *http.Request) (*http.Request, error) {
	// clones the request
	r2 := r.Clone(context.Background())

	// r.Clone just copies the io.Reader, which means whoever reads first wins it
	// Therefore we need to duplicate the body manually
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = ioutil.NopCloser(bytes.NewReader(body))
	r2.Body = ioutil.NopCloser(bytes.NewReader(body))

	return r2, nil
}
