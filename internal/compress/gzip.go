package compress

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
)

const (
	gzipEncoding = "gzip"
)

type GzipWriteEngine struct{}

func NewGzipWriteEngine() *GzipWriteEngine {
	return &GzipWriteEngine{}
}

func (gwe *GzipWriteEngine) Name() string {
	return gzipEncoding
}

func (gwe *GzipWriteEngine) Applicable(header http.Header) bool {
	return checkAcceptEncodingIncludes(header, gzipEncoding)
}

func (gwe *GzipWriteEngine) NewResponseWriter(w http.ResponseWriter, level int) (CompressedResponseWriter, error) {
	gw, err := gzip.NewWriterLevel(w, normalizeGzipLevel(level))
	if err != nil {
		return nil, fmt.Errorf("error intitalizing gzip writer: %w", err)
	}
	return newGzipResponseWriter(w, gw), nil
}

func (gwe *GzipWriteEngine) WriteFlushed(data []byte, level int) ([]byte, error) {
	var b bytes.Buffer
	gw, err := gzip.NewWriterLevel(&b, normalizeGzipLevel(level))
	if err != nil {
		return nil, fmt.Errorf("error intitalizing gzip writer: %w", err)
	}
	_, err = gw.Write(data)
	if err != nil {
		return nil, fmt.Errorf("error writing to gzip writer: %w", err)
	}
	err = gw.Close()
	if err != nil {
		return nil, fmt.Errorf("error compressing data with gzip: %w", err)
	}
	return b.Bytes(), nil
}

func (gwe *GzipWriteEngine) SetContentEncoding(header http.Header) {
	header.Set("Content-Encoding", gzipEncoding)
}

type GzipReadEngine struct{}

func NewGzipReadEngine() *GzipReadEngine {
	return &GzipReadEngine{}
}

func (gre *GzipReadEngine) Name() string {
	return gzipEncoding
}

func (gre *GzipReadEngine) Applicable(header http.Header) bool {
	return checkContentEncoding(header, gzipEncoding)
}

func (gre *GzipReadEngine) ReadAll(r io.Reader) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("error initializing gzip reader: %w", err)
	}

	data, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("error reading gzipped data: %w", err)
	}
	return data, nil
}

func normalizeGzipLevel(level int) int {
	if level <= 0 {
		return gzip.DefaultCompression
	}
	if level > gzip.BestCompression {
		return gzip.BestCompression
	}
	return level
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gw      *gzip.Writer
	written bool
}

func newGzipResponseWriter(w http.ResponseWriter, gw *gzip.Writer) *gzipResponseWriter {
	return &gzipResponseWriter{
		ResponseWriter: w,
		gw:             gw,
	}
}

func (grw *gzipResponseWriter) Write(data []byte) (int, error) {
	if checkContentTypeForCompression(grw.ResponseWriter.Header()) {
		bcnt, err := grw.gw.Write(data)
		if err == nil {
			grw.written = true
		}
		return bcnt, err
	} else {
		return grw.ResponseWriter.Write(data)
	}
}

func (grw *gzipResponseWriter) WriteHeader(statusCode int) {
	if checkContentTypeForCompression(grw.ResponseWriter.Header()) {
		grw.ResponseWriter.Header().Del("Content-Encoding")
		grw.ResponseWriter.Header().Add("Content-Encoding", gzipEncoding)
	}
	grw.ResponseWriter.WriteHeader(statusCode)
}

func (grw *gzipResponseWriter) Close() error {
	if !grw.written {
		return nil
	}
	return grw.gw.Close()
}
