package relay

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pyroscope-io/pyroscope/pkg/storage/segment"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Address   string
	AuthToken string
	Tags      map[string]string
}

type Relay struct {
	log    *logrus.Logger
	config Config
	client *http.Client
}

func NewRelay() *Relay {
	return &Relay{}
}

func (t Relay) StartServer() error {
	mux := http.NewServeMux()
	mux.Handle("/", t)

	server := &http.Server{
		Handler: mux,
		Addr:    "0.0.0.0:4040",
	}

	fmt.Println("starting server")
	return server.ListenAndServe()
}

// ServeHTTP requests shadows traffic to the remote server
// Then offloads to the original handler
func (t Relay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r2, err := t.cloneRequest(r)
	if err != nil {
		t.log.Error("Failed to clone request", err)
		return
	}

	// TODO(eh-am): put immediately in a queue and process later?
	t.log.Debugf("Sending to remote")
	t.sendToRemote(w, r2)

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

	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}

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

// TODO
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
