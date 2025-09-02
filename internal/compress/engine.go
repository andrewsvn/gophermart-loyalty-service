package compress

import (
	"io"
	"net/http"
	"strings"
)

var compressContentTypes = []string{
	"application/json",
	"text/html",
}

type CompressedResponseWriter interface {
	http.ResponseWriter
	Close() error
}

type WriteEngine interface {
	Name() string

	// Applicable determines if engine can be used to write response body
	Applicable(header http.Header) bool
	// NewResponseWriter returns wrapped ResponseWriter to use in underlying handlers
	NewResponseWriter(w http.ResponseWriter, level int) (CompressedResponseWriter, error)

	// WriteFlushed compresses source data into new buffer and closes the writer to finalize write operation
	WriteFlushed(data []byte, level int) ([]byte, error)
	// SetContentEncoding adds an http header to indicate that content is compressed by a certain algorithm
	SetContentEncoding(header http.Header)
}

type ReadEngine interface {
	Name() string

	// Applicable determines if engine can be used to read request body
	Applicable(header http.Header) bool
	// ReadAll extracts all data from reader decompressing it by underlying algorithm
	ReadAll(r io.Reader) ([]byte, error)
}

func checkAcceptEncodingIncludes(header http.Header, encodings ...string) bool {
	acceptEncoding := header.Get("Accept-Encoding")
	for _, acceptable := range strings.Split(acceptEncoding, ",") {
		for _, encoding := range encodings {
			if encoding == strings.TrimSpace(acceptable) {
				return true
			}
		}
	}
	return false
}

func checkContentEncoding(header http.Header, encodings ...string) bool {
	contentEncoding := header.Get("Content-Encoding")
	for _, encoding := range encodings {
		if encoding == contentEncoding {
			return true
		}
	}
	return false
}

func checkContentTypeForCompression(header http.Header) bool {
	contentType := header.Get("Content-Type")
	contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	for _, allowed := range compressContentTypes {
		if allowed == contentType {
			return true
		}
	}
	return false
}
