package relay_test

import (
	"context"
	"net/http"
	"testing"
)

func TestRelay(t *testing.T) {

}

type mockStartStopper struct {
	start func() error
	stop  func() error
}

func (mss *mockStartStopper) Start() error                   { return mss.start() }
func (mss *mockStartStopper) Stop(ctx context.Context) error { return mss.stop() }

type mockRelayer struct {
	//	*mockStartStopper
	relay func(*http.Request) error
	start func() error
	stop  func() error
}

func (mr *mockRelayer) Send(req *http.Request) error   { return mr.relay(req) }
func (mr *mockRelayer) Start() error                   { return mr.start() }
func (mr *mockRelayer) Stop(ctx context.Context) error { return mr.stop() }

//func TestRelayTimeout(t *testing.T) {
//	logger := noopLogger()
//
//	mss := &mockStartStopper{
//		start: func() error {
//			return nil
//		},
//		stop: func() error {
//			//			time.Sleep(10 * time.Second)
//			return nil
//		},
//	}
//
//	mr := &mockRelayer{
//		start: func() error {
//			return nil
//		},
//		stop: func() error {
//			//			time.Sleep(10 * time.Second)
//			return nil
//		},
//		relay: func(r *http.Request) error {
//			return nil
//		},
//	}
//
//	relay2 := relay.NewRelay2(logger, &relay.Relay2Cfg{
//		ShutdownTimeout: 10 * time.Millisecond,
//	}, mss, mr)
//
//	relay2.Start()
//	err := relay2.Stop()
//	assert.Error(t, err)
//}
