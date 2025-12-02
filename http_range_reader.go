package gocog

import (
	"fmt"
	"io"
	"sync"

	"github.com/valyala/fasthttp"
)

// HTTPRangeReader implements io.ReadSeeker for reading files via HTTP range requests
type HTTPRangeReader struct {
	url    string
	client *fasthttp.Client
	size   int64
	mu     sync.Mutex
	pos    int64
}

// NewHTTPRangeReader creates a new HTTP range reader
func NewHTTPRangeReader(url string, client *fasthttp.Client) *HTTPRangeReader {
	rr := &HTTPRangeReader{
		url:    url,
		client: client,
		pos:    0,
	}

	// Get file size
	rr.size = rr.getSize()

	return rr
}

// getSize gets the file size using HEAD request
func (rr *HTTPRangeReader) getSize() int64 {
	if rr.client == nil {
		return -1
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(rr.url)
	req.Header.SetMethod("HEAD")

	err := rr.client.Do(req, resp)
	if err != nil {
		return -1
	}

	contentLength := resp.Header.ContentLength()
	if contentLength > 0 {
		return int64(contentLength)
	}

	return -1
}

// Read reads data from the current position
func (rr *HTTPRangeReader) Read(p []byte) (n int, err error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if rr.pos >= rr.size && rr.size > 0 {
		return 0, io.EOF
	}

	// Calculate how much to read
	toRead := len(p)
	if rr.size > 0 && rr.pos+int64(toRead) > rr.size {
		toRead = int(rr.size - rr.pos)
	}

	// Make range request
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(rr.url)
	req.Header.SetMethod("GET")

	// Set range header
	rangeEnd := rr.pos + int64(toRead) - 1
	if rr.size > 0 && rangeEnd >= rr.size {
		rangeEnd = rr.size - 1
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rr.pos, rangeEnd))

	err = rr.client.Do(req, resp)
	if err != nil {
		return 0, err
	}

	statusCode := resp.StatusCode()
	if statusCode != fasthttp.StatusPartialContent && statusCode != fasthttp.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", statusCode)
	}

	// Read data
	body := resp.Body()
	if len(body) < toRead {
		toRead = len(body)
	}
	n = copy(p[:toRead], body)
	if n < toRead && n < len(p) {
		err = io.ErrUnexpectedEOF
	} else if n == 0 && len(body) == 0 {
		err = io.EOF
	}
	if err == io.ErrUnexpectedEOF {
		err = nil // Partial read is OK
	}

	rr.pos += int64(n)
	return n, err
}

// Seek sets the offset for the next Read
func (rr *HTTPRangeReader) Seek(offset int64, whence int) (int64, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = rr.pos + offset
	case io.SeekEnd:
		if rr.size < 0 {
			return 0, fmt.Errorf("cannot seek from end: file size unknown")
		}
		newPos = rr.size + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newPos < 0 {
		return 0, fmt.Errorf("negative position: %d", newPos)
	}

	rr.pos = newPos
	return rr.pos, nil
}

// Size returns the file size, or -1 if unknown
func (rr *HTTPRangeReader) Size() int64 {
	return rr.size
}
