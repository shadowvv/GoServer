package webService

import (
	"context"
	"errors"
	"net/http"
	"time"
)

const defaultWorkerLimit = 1024

type HttpConfig struct {
	Address        string        `yaml:"address"`
	ReadTimeoutMs  time.Duration `yaml:"readTimeout"`
	WriteTimeoutMs time.Duration `yaml:"writeTimeout"`
	IdleTimeoutMs  time.Duration `yaml:"idleTimeout"`
	MaxBodySize    int64         `yaml:"maxBodySize"`
}

type HttpWebService struct {
	addr            string
	srv             *http.Server
	mux             *http.ServeMux
	defaultMaxBytes int64
	workerLimiter   chan struct{}
}

func NewHttpWebService(cfg *HttpConfig) *HttpWebService {
	mux := http.NewServeMux()

	s := &HttpWebService{
		addr:            cfg.Address,
		mux:             mux,
		defaultMaxBytes: cfg.MaxBodySize,
		workerLimiter:   make(chan struct{}, defaultWorkerLimit),
	}

	var handler http.Handler = mux
	handler = corsMiddleware(handler)

	s.srv = &http.Server{
		Addr:         cfg.Address,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeoutMs,
		WriteTimeout: cfg.WriteTimeoutMs,
		IdleTimeout:  cfg.IdleTimeoutMs,
	}
	return s
}

func maxBytesMiddleware(next http.HandlerFunc, maxBytes int64) http.HandlerFunc {
	if maxBytes <= 0 {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next(w, r)
	}
}

func limitMiddleware(next http.HandlerFunc, workerLimiter chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case workerLimiter <- struct{}{}:
			defer func() { <-workerLimiter }()
			next(w, r)
		default:
			http.Error(w, "server busy", http.StatusTooManyRequests)
		}
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *HttpWebService) RegisterRoutes(path string, handler http.HandlerFunc) {
	s.mux.HandleFunc(path, limitMiddleware(maxBytesMiddleware(handler, s.defaultMaxBytes), s.workerLimiter))
}

func (s *HttpWebService) RegisterRoutesWithMaxBody(path string, handler http.HandlerFunc, maxBytes int64) {
	s.mux.HandleFunc(path, limitMiddleware(maxBytesMiddleware(handler, maxBytes), s.workerLimiter))
}

func (s *HttpWebService) Start() {
	err := s.srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func (s *HttpWebService) Shutdown(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}
