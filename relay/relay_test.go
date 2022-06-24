package relay_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/pyroscope-io/pyroscope-lambda-extension/relay"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var logger *logrus.Entry

func init() {
	logger = noopLogger()
}

func TestRelay(t *testing.T) {

	endpoint := "/ingest?aggregationType=sum&from=1655819920&name=simple.golang.app-new%7B%7D&sampleRate=100&spyName=gospy&units=samples&until=1655819927"
	u, err := url.Parse(endpoint)
	assert.NoError(t, err)
	profile := readTestdataFile(t, "testdata/profile.pprof")
	authorizationHeader := "Bearer 123"

	remoteServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, u.Path, r.URL.Path, "path is mirrored")
			assert.Equal(t, u.RawQuery, r.URL.RawQuery, "query params are mirrored")

			body := &bytes.Buffer{}
			body.ReadFrom(r.Body)
			assert.Equal(t, profile, body.Bytes(), "body is mirrored")

			assert.Equal(t, authorizationHeader, r.Header.Get("Authorization"), "auth header is mirrored")
			// TODO(eh-am): add wait group?
		}),
	)

	r := relay.NewRelay(&relay.Config{Address: remoteServer.URL}, logger)

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(profile))
	req.Header.Set("Authorization", authorizationHeader)
	assert.NoError(t, err)

	r.ServeHTTP(httptest.NewRecorder(), req)
}

// This tests checks that upon shutdown, we wait for the job queue to drain
func TestShutdown(t *testing.T) {
	logger := logrus.New().WithFields(logrus.Fields{})
	logrus.SetLevel(logrus.TraceLevel)

	var wg sync.WaitGroup

	// Simulate a slow remote server
	remoteServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			wg.Done()
			time.Sleep(time.Millisecond * 100)
			w.WriteHeader(200)
		}),
	)

	// Create a relay server
	r := relay.NewRelay(&relay.Config{Address: remoteServer.URL, ServerAddress: ":0"}, logger)

	req, err := http.NewRequest(http.MethodPost, "/", nil)
	assert.NoError(t, err)

	// Start the server
	go func() {
		err := r.Start()
		assert.NoError(t, err)
	}()

	wg.Add(1)
	r.ServeHTTP(httptest.NewRecorder(), req)

	// Wait for the request to be processed
	wg.Wait()

	err = r.Stop()
	assert.NoError(t, err)
}

func noopLogger() *logrus.Entry {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	return logger.WithFields(logrus.Fields{})
}

func readTestdataFile(t *testing.T, name string) []byte {
	f, err := ioutil.ReadFile(name)
	assert.NoError(t, err)
	return f
}
