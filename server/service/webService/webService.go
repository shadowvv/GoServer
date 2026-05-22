package webService

import (
	"golang.org/x/net/context"
	"net/http"
	"time"
)

var workerLimiter = make(chan struct{}, 1024) // 最大并发 1024

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
}

func NewHttpWebService(cfg *HttpConfig) *HttpWebService {
	mux := http.NewServeMux()

	s := &HttpWebService{
		addr:            cfg.Address,
		mux:             mux,
		defaultMaxBytes: cfg.MaxBodySize,
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

// maxBytesMiddleware 在请求进入业务 handler 前限制 Body 大小，maxBytes <= 0 表示不限制。
func maxBytesMiddleware(next http.HandlerFunc, maxBytes int64) http.HandlerFunc {
	if maxBytes <= 0 {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next(w, r)
	}
}

func limitMiddleware(next http.HandlerFunc) http.HandlerFunc {
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
	s.mux.HandleFunc(path, limitMiddleware(maxBytesMiddleware(handler, s.defaultMaxBytes)))
}

// RegisterRoutesWithMaxBody 为单条路由设置独立的 Body 上限，maxBytes <= 0 表示不限制。
func (s *HttpWebService) RegisterRoutesWithMaxBody(path string, handler http.HandlerFunc, maxBytes int64) {
	s.mux.HandleFunc(path, limitMiddleware(maxBytesMiddleware(handler, maxBytes)))
}

func (s *HttpWebService) Start() {
	err := s.srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func (s *HttpWebService) Shutdown(ctx context.Context) error {
	if s.srv != nil {
		_ = s.srv.Shutdown(ctx)
	}
	return nil
}
