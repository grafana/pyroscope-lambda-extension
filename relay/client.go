package relay

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pyroscope-io/pyroscope-lambda-extension/internal/sessionid"
)

var (
	ErrMakingRequest = errors.New("failed to make request")
	ErrNotOkResponse = errors.New("response not ok")
)

type RemoteClientCfg struct {
	// Address refers to the remote address the request will be made to
	Address             string
	AuthToken           string
	BasicAuthUser       string
	BasicAuthPassword   string
	TenantID            string
	HTTPHeadersJSON     string
	Timeout             time.Duration
	MaxIdleConnsPerHost int
	SessionID           string
}

type RemoteClient struct {
	config    *RemoteClientCfg
	client    *http.Client
	headers   map[string]string
	log       *logrus.Entry
	sessionID string
}

func NewRemoteClient(log *logrus.Entry, config *RemoteClientCfg) *RemoteClient {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = time.Second * 10
	}
	if config.MaxIdleConnsPerHost == 0 {
		config.MaxIdleConnsPerHost = 5
	}
	headers := make(map[string]string)
	if config.HTTPHeadersJSON != "" {
		err := json.Unmarshal([]byte(config.HTTPHeadersJSON), &headers)
		if err != nil {
			log.Error(fmt.Errorf("failed to parse headers json %w", err))
		}
	}
	return &RemoteClient{
		log:       log,
		config:    config,
		sessionID: config.SessionID,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
			},
		},
	}
}

// Send relays the request to the remote server
func (r *RemoteClient) Send(req *http.Request) error {
	if req.Body != nil {
		defer req.Body.Close()
	}
	r.enhanceWithAuthToken(req)
	if r.config.TenantID != "" {
		req.Header.Set("X-Scope-OrgID", r.config.TenantID)
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	host := r.config.Address

	u, _ := url.Parse(host)

	req.RequestURI = ""
	req.URL.Host = u.Host
	req.URL.Scheme = u.Scheme
	req.URL.User = u.User
	req.URL.Path = path.Join(u.Path, req.URL.Path)
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = u.Host
	sessionid.InjectToRequest(r.sessionID, req)
	// TODO(eh-am): check it's a request to /ingest?
	r.log.Debugf("Making request to %s", req.URL.String())
	res, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMakingRequest, err)
	}
	defer res.Body.Close()

	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		respBody, _ := io.ReadAll(res.Body)
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
	} else if r.config.BasicAuthUser != "" && r.config.BasicAuthPassword != "" {
		req.SetBasicAuth(r.config.BasicAuthUser, r.config.BasicAuthPassword)
	}
}
