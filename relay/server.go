package relay

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
)

type ServerCfg struct {
	ServerAddress string
}

type Server struct {
	config *ServerCfg
	log    *logrus.Entry
	server *http.Server
}

func NewServer(logger *logrus.Entry, config *ServerCfg, handlerFunc http.HandlerFunc) *Server {
	mux := http.NewServeMux()
	svr := &http.Server{
		Handler: mux,
		Addr:    config.ServerAddress,
	}

	server := &Server{
		config: config,
		log:    logger,
		server: svr,
	}

	mux.Handle("/", handlerFunc)
	return server
}

// Start starts serving requests, this is a blocking operation
func (s *Server) Start() error {
	s.log.Debugf("Serving on %s", s.config.ServerAddress)
	err := s.server.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
