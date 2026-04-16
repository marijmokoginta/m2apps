package api

import (
	"context"
	"fmt"
	"m2apps/internal/progress"
	"m2apps/internal/storage"
	"net"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(store storage.Storage, progressManager *progress.Manager, accessLog func(string)) *Server {
	handler := &Handler{
		Store:     store,
		Progress:  progressManager,
		AccessLog: accessLog,
	}

	return &Server{
		httpServer: &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

func (s *Server) Serve(listener net.Listener) error {
	if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("api server failed: %w", err)
	}
	return nil
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}
