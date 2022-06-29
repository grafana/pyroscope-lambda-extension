package relay

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	ErrMakingRequest = errors.New("failed to make request")
	ErrNotOkResponse = errors.New("response not ok")
)

type RemoteClientCfg struct {
	// Address refers to the remote address the request will be made to
	Address   string
	AuthToken string
	Timeout   time.Duration
}

type RemoteClient struct {
	config *RemoteClientCfg
	client *http.Client
	log    *logrus.Entry
}

func NewRemoteClient(log *logrus.Entry, config *RemoteClientCfg) *RemoteClient {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = time.Second * 10
	}

	return &RemoteClient{
		log:    log,
		config: config,
		// TODO(eh-am): improve this client with timeouts and whatnot
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Send relays the request to the remote server
func (r *RemoteClient) Send(req *http.Request) error {
	if req.Body != nil {
		defer req.Body.Close()
	}
	r.enhanceWithAuthToken(req)

	host := r.config.Address

	u, _ := url.Parse(host)

	req.RequestURI = ""
	req.URL.Host = u.Host
	req.URL.Scheme = u.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = u.Host

	// TODO(eh-am): check it's a request to /ingest?
	r.log.Debugf("Making request to %s", req.URL.String())
	res, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMakingRequest, err)
	}
	defer res.Body.Close()

	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		respBody, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("%w: %v", ErrNotOkResponse, fmt.Errorf("status code: '%d'. body: '%s'", res.StatusCode, respBody))
	}

	return nil
}

// enhanceWithAuthToken adds an Authorization header if an AuthToken is supplied
// note that if no authToken is set, it's possible that the Authorization header
// from the original request is kept
func (r *RemoteClient) enhanceWithAuthToken(req *http.Request) {
	token := r.config.AuthToken

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}
