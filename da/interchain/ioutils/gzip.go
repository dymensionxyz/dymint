package ioutils

import (
	"bytes"
	"compress/gzip"
	"io"
)

// Note: []byte can never be const as they are inherently mutable
var (
	// magic bytes to identify gzip.
	// See https://www.ietf.org/rfc/rfc1952.txt
	// and https://github.com/golang/go/blob/master/src/net/http/sniff.go#L186
	gzipIdent = []byte("\x1F\x8B\x08")
)

// IsGzip returns checks if the file contents are gzip compressed
func IsGzip(input []byte) bool {
	return len(input) >= 3 && bytes.Equal(gzipIdent, input[0:3])
}

// Gzip compresses the input ([]byte)
func Gzip(input []byte) ([]byte, error) {
	// Create gzip writer
	var b bytes.Buffer
	w := gzip.NewWriter(&b)

	_, err := w.Write(input)
	if err != nil {
		return nil, err
	}

	// You must close this first to flush the bytes to the buffer
	err = w.Close()
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// Gunzip decompresses the input ([]byte)
func Gunzip(input []byte) ([]byte, error) {
	// Create gzip reader
	b := bytes.NewReader(input)
	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}

	output, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = r.Close()
	if err != nil {
		return nil, err
	}

	return output, nil
}
