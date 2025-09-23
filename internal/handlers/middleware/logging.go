package middleware

import (
	"net/http"
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"go.uber.org/zap"
)

type HTTPLogging struct {
	logger *zap.SugaredLogger
}

func NewHTTPLogging(l *zap.Logger) *HTTPLogging {
	return &HTTPLogging{
		logger: logging.ComponentLogger(l, "http-logging"),
	}
}

type enrichedResponseWriter struct {
	http.ResponseWriter

	ResponseStatus int
	ResponseSize   int
}

func newEnrichedResponseWriter(w http.ResponseWriter) *enrichedResponseWriter {
	return &enrichedResponseWriter{
		ResponseWriter: w,
	}
}

func (ew *enrichedResponseWriter) WriteHeader(status int) {
	ew.ResponseStatus = status
	ew.ResponseWriter.WriteHeader(status)
}

func (ew *enrichedResponseWriter) Write(b []byte) (int, error) {
	written, err := ew.ResponseWriter.Write(b)
	ew.ResponseSize += written
	return ew.ResponseSize, err
}

func (l *HTTPLogging) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func(logger *zap.SugaredLogger) {
			_ = logger.Sync()
		}(l.logger)

		l.logger.Infow("Request received",
			"url", r.URL.String(),
			"method", r.Method,
			"content-type", r.Header.Get("Content-Type"),
			"content-encoding", r.Header.Get("Content-Encoding"))

		ew := newEnrichedResponseWriter(w)
		start := time.Now()
		next.ServeHTTP(ew, r)
		duration := time.Since(start)

		l.logger.Infow("Request processed",
			"url", r.URL.String(),
			"method", r.Method,
			"status", ew.ResponseStatus,
			"responseSize", ew.ResponseSize,
			"content-type", ew.Header().Get("Content-Type"),
			"content-encoding", ew.Header().Get("Content-Encoding"),
			"duration", duration.String())
	})
}
