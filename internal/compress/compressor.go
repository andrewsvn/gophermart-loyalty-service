package compress

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

type Compressor struct {
	engines []WriteEngine
	logger  *zap.SugaredLogger
}

func NewCompressor(logger *zap.Logger, engines ...WriteEngine) *Compressor {
	cmpLogger := logger.Sugar().With(zap.String("component", "compressor"))
	return &Compressor{
		engines: engines,
		logger:  cmpLogger,
	}
}

func (c *Compressor) CreateCompressWriter(w http.ResponseWriter, r *http.Request) (CompressedResponseWriter, error) {
	for _, wEngine := range c.engines {
		if wEngine.Applicable(r.Header) {
			c.logger.Debug(
				"Compression engine chosen for response writing",
				zap.String("name", wEngine.Name()),
			)
			crw, err := wEngine.NewResponseWriter(w, 0)
			if err != nil {
				return nil, fmt.Errorf("error creating compress writer: %w", err)
			}
			return crw, nil
		}
	}
	return nil, nil
}
