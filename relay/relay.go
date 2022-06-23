package relay

import (
	"context"
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
}

type Relay struct {
	log    *logrus.Entry
	config *Config
	client *http.Client

	server *http.Server
	wg     sync.WaitGroup
}

func NewRelay(config *Config, logger *logrus.Entry) *Relay {
	logger = logrus.WithFields(logrus.Fields{"svc": svcName})

	return &Relay{
		config: config,
		log:    logger,
		// TODO(eh-am): improve this client
		client: &http.Client{},
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

	t.log.Debugf("Serving on %s", addr)
	err := t.server.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (t *Relay) Stop() error {
	// https://docs.aws.amazon.com/lambda/latest/dg/runtimes-extensions-api.html#runtimes-lifecycle-shutdown
	shutdownLimit := time.Second * 2

	ctx, cancel := context.WithTimeout(context.Background(), shutdownLimit)
	defer cancel()

	t.log.Debug("Shutting down with timeout of ", shutdownLimit)
	t.wg.Wait()

	// TODO(eh-am): wait for the inflight requests?
	return t.server.Shutdown(ctx)
}

// ServeHTTP requests shadows traffic to the remote server
func (t *Relay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//	t.wg.Add(1)
	//	defer t.wg.Done()

	// TODO(eh-am):
	// * add to a queue
	// * respond immediately
	//	go func() {
	//		// TODO(eh-am): put immediately in a queue and process later?
	t.log.Trace("Sending to remote")
	t.sendToRemote(w, r)
	t.log.Trace("Sent to remote")
	//	}()

	// TODO(eh-am): respond
	w.WriteHeader(200)
}

func (t *Relay) sendToRemote(_ http.ResponseWriter, r *http.Request) {
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
		t.log.Error("Failed to shadow request. Dropping it", err)
		return
	}

	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		// TODO(eh-am): print the error message if there's any?
		t.log.Errorf("Request to remote failed with statusCode: '%d'", res.StatusCode)
	}
}
