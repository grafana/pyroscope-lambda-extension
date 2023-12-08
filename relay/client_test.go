package relay_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pyroscope-io/pyroscope-lambda-extension/internal/flameql"
	"github.com/pyroscope-io/pyroscope-lambda-extension/internal/sessionid"
	"github.com/pyroscope-io/pyroscope-lambda-extension/relay"
)

func TestRemoteClient(t *testing.T) {
	logger := noopLogger()

	endpoint := "/ingest?aggregationType=sum&from=1655819920&name=simple.golang.app-new%7B%7D&sampleRate=100&spyName=gospy&units=samples&until=1655819927"
	u, err := url.Parse(endpoint)
	assert.NoError(t, err)
	profile := readTestdataFile(t, "testdata/profile.pprof")
	authToken := "123"

	rcc := &relay.RemoteClientCfg{
		AuthToken: "123",
		SessionID: sessionid.New().String(),
	}

	remoteServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, u.Path, r.URL.Path, "path is mirrored")

			q := r.URL.Query()
			parsed, err := flameql.ParseKey(q.Get("name"))
			require.NoError(t, err)
			require.Equal(t,
				rcc.SessionID,
				parsed.Labels()[sessionid.LabelName],
				"requests has __session_id__ label")

			delete(parsed.Labels(), sessionid.LabelName)
			q.Set("name", parsed.Normalized())
			assert.Equal(t, u.RawQuery, q.Encode(), "query params are mirrored")

			body := &bytes.Buffer{}
			body.ReadFrom(r.Body)
			assert.Equal(t, profile, body.Bytes(), "body is mirrored")

			assert.Equal(t, "Bearer "+authToken, r.Header.Get("Authorization"), "auth header is mirrored")
		}),
	)

	rcc.Address = remoteServer.URL
	remoteClient := relay.NewRemoteClient(logger, rcc)

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(profile))
	assert.NoError(t, err)

	err = remoteClient.Send(req)
	assert.NoError(t, err)
}

func TestRemoteClientNon2xxError(t *testing.T) {
	logger := noopLogger()

	endpoint := "/ingest?aggregationType=sum&from=1655819920&name=simple.golang.app-new%7B%7D&sampleRate=100&spyName=gospy&units=samples&until=1655819927"

	remoteServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		}),
	)

	remoteClient := relay.NewRemoteClient(logger, &relay.RemoteClientCfg{Address: remoteServer.URL})

	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	assert.NoError(t, err)

	err = remoteClient.Send(req)
	assert.ErrorIs(t, err, relay.ErrNotOkResponse)
}

func TestRemoteClientIncompleteRequestError(t *testing.T) {
	logger := noopLogger()

	invalidUrl := ""
	remoteClient := relay.NewRemoteClient(logger, &relay.RemoteClientCfg{Address: invalidUrl})

	req, err := http.NewRequest(http.MethodPost, "/", nil)
	assert.NoError(t, err)

	// There should an error
	err = remoteClient.Send(req)
	assert.ErrorIs(t, err, relay.ErrMakingRequest)
}

func TestRemoteClientTimeout(t *testing.T) {
	logger := noopLogger()

	endpoint := "/ingest"

	remoteServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Second * 999)
			w.WriteHeader(200)
		}),
	)

	remoteClient := relay.NewRemoteClient(logger, &relay.RemoteClientCfg{
		Address: remoteServer.URL,
		Timeout: time.Millisecond * 50,
	})

	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	assert.NoError(t, err)

	err = remoteClient.Send(req)
	assert.ErrorIs(t, err, relay.ErrMakingRequest)
}
