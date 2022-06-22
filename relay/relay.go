package relay

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pyroscope-io/pyroscope/pkg/storage/segment"
	"github.com/sirupsen/logrus"
)

// Service Name used in logs
const svcName = "pyroscope-lambda-ext-relay"

type Config struct {
	Address   string
	AuthToken string
	Tags      map[string]string

	ServerAddress string
}

type Relay struct {
	log    *logrus.Entry
	config *Config
	client *http.Client

	server *http.Server
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

func (t Relay) Stop() error {
	// https://docs.aws.amazon.com/lambda/latest/dg/runtimes-extensions-api.html#runtimes-lifecycle-shutdown
	shutdownLimit := time.Second * 2

	ctx, cancel := context.WithTimeout(context.Background(), shutdownLimit)
	defer cancel()

	// TODO(eh-am): wait for the inflight requests?

	return t.server.Shutdown(ctx)
}

// ServeHTTP requests shadows traffic to the remote server
func (t Relay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//	t.log.Trace("Cloning request")
	//	r2, err := t.cloneRequest(r)
	//	if err != nil {
	//		t.log.Error("Failed to clone request", err)
	//		return
	//	}
	//
	// TODO(eh-am): put immediately in a queue and process later?
	t.log.Trace("Sending to remote")
	t.sendToRemote(w, r)

	// TODO(eh-am): respond
}

func (Relay) cloneRequest(r *http.Request) (*http.Request, error) {
	// clones the request
	r2 := r.Clone(r.Context())

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

func (t Relay) sendToRemote(_ http.ResponseWriter, r *http.Request) {
	host := t.config.Address
	token := t.config.AuthToken

	u, _ := url.Parse(host)

	r.RequestURI = ""
	r.URL.Host = u.Host
	r.URL.Scheme = u.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = u.Host

	// needs to happen after
	t.enhanceWithTags(r)

	// TODO(eh-am): token could be setup in the proxy directly
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}

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

// TODO(eh-am): how to receive these tags?
func (t Relay) enhanceWithTags(r *http.Request) {
	appName := r.URL.Query().Get("name")

	if appName == "" {
		t.log.Errorf("Expected to find queryParam 'name' but found nothing. Could not add tags to request")
		return
	}

	k, err := segment.ParseKey(appName)
	if err != nil {
		t.log.Errorf("Failed to parse key from app name. Could not add tags to request")
		return
	}

	for tag, value := range t.config.Tags {
		k.Labels()[tag] = value
	}

	logrus.Debug("enhancing with tags", k.Normalized())
	params := r.URL.Query()
	params.Set("name", k.Normalized())
	r.URL.RawQuery = params.Encode()
}
