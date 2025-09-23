package compress

import (
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type Decompressor struct {
	engines []ReadEngine
	logger  *zap.SugaredLogger
}

func NewDecompressor(logger *zap.Logger, engines ...ReadEngine) *Decompressor {
	dcmpLogger := logger.Sugar().With(zap.String("component", "decompressor"))
	return &Decompressor{
		engines: engines,
		logger:  dcmpLogger,
	}
}

func (d *Decompressor) ReadRequestBody(r *http.Request) ([]byte, error) {
	for _, engine := range d.engines {
		if engine.Applicable(r.Header) {
			d.logger.Debug(
				"Decompress reader chosen for request body",
				zap.String("name", engine.Name()),
			)
			b, err := engine.ReadAll(r.Body)
			if err != nil {
				return nil, fmt.Errorf("can't decompress request body: %w", err)
			}
			return b, nil
		}
	}
	b, err := io.ReadAll(r.Body)
	return b, err
}
