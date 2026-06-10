package handler

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const opsObservedPayloadLimit = 64 * 1024

type opsObservedBodyCapture struct {
	src           io.ReadCloser
	limit         int
	contentLength int64
	buf           bytes.Buffer
	reader        io.Reader
	servedBytes   int
	truncated     bool
}

func installOpsObservedRequestBody(c *gin.Context) *opsObservedBodyCapture {
	if c == nil || c.Request == nil || c.Request.Body == nil || c.Request.Body == http.NoBody {
		return nil
	}
	capture := &opsObservedBodyCapture{
		src:           c.Request.Body,
		limit:         opsObservedPayloadLimit,
		contentLength: c.Request.ContentLength,
	}
	if err := capture.prefetch(); err != nil {
		log.Printf("[OpsErrorLogger] capture request body failed: %v", err)
	}
	cloned := c.Request.Clone(c.Request.Context())
	cloned.Body = capture
	c.Request = cloned
	return capture
}

func (b *opsObservedBodyCapture) prefetch() error {
	maxRead := int64(b.limit)
	if b.contentLength >= 0 && b.contentLength <= int64(b.limit) {
		maxRead = b.contentLength
	}
	if b.contentLength < 0 {
		maxRead = int64(b.limit) + 1
	}
	if maxRead <= 0 {
		b.reader = b.src
		return nil
	}
	prefix, err := io.ReadAll(io.LimitReader(b.src, maxRead))
	if err != nil {
		b.reader = io.MultiReader(bytes.NewReader(prefix), b.src)
		return err
	}
	b.storePrefix(prefix)
	b.reader = io.MultiReader(bytes.NewReader(prefix), b.src)
	return nil
}

func (b *opsObservedBodyCapture) storePrefix(prefix []byte) {
	if len(prefix) > b.limit {
		_, _ = b.buf.Write(prefix[:b.limit])
		b.truncated = true
		return
	}
	_, _ = b.buf.Write(prefix)
	if b.contentLength > int64(b.limit) {
		b.truncated = true
	}
}

func (b *opsObservedBodyCapture) Read(p []byte) (int, error) {
	if b.reader == nil {
		b.reader = b.src
	}
	n, err := b.reader.Read(p)
	if n > 0 {
		b.servedBytes += n
	}
	return n, err
}

func (b *opsObservedBodyCapture) Close() error {
	return b.src.Close()
}

func (b *opsObservedBodyCapture) bodyString() string {
	if b == nil || b.buf.Len() == 0 {
		return ""
	}
	return b.buf.String()
}

func (b *opsObservedBodyCapture) bodyBytes() (int, bool) {
	if b == nil {
		return 0, false
	}
	if b.contentLength > 0 {
		return int(b.contentLength), true
	}
	if b.servedBytes > 0 {
		return b.servedBytes, true
	}
	if b.buf.Len() > 0 && !b.truncated {
		return b.buf.Len(), true
	}
	return 0, false
}

func (b *opsObservedBodyCapture) isTruncated() bool {
	if b == nil {
		return false
	}
	return b.truncated
}

func applyOpsObservedHTTPDetails(c *gin.Context, entry *service.OpsInsertErrorLogInput, reqBody *opsObservedBodyCapture, w *opsCaptureWriter) {
	if c == nil || entry == nil {
		return
	}
	applyOpsObservedHeaders(c, entry)
	applyOpsObservedRequestBody(entry, reqBody)
	applyOpsObservedResponseBody(entry, w)
}

func applyOpsObservedHeaders(c *gin.Context, entry *service.OpsInsertErrorLogInput) {
	if c.Request != nil {
		headers, err := service.SanitizeOpsObservedHeaders(c.Request.Header)
		if err != nil {
			log.Printf("[OpsErrorLogger] sanitize request headers failed: %v", err)
		} else {
			entry.ObservedRequestHeadersJSON = headers
		}
	}
	headers, err := service.SanitizeOpsObservedHeaders(c.Writer.Header())
	if err != nil {
		log.Printf("[OpsErrorLogger] sanitize response headers failed: %v", err)
		return
	}
	entry.ObservedResponseHeadersJSON = headers
}

func applyOpsObservedRequestBody(entry *service.OpsInsertErrorLogInput, body *opsObservedBodyCapture) {
	if body == nil {
		return
	}
	entry.ObservedRequestBody = body.bodyString()
	entry.ObservedRequestBodyTruncated = body.isTruncated()
	if n, ok := body.bodyBytes(); ok {
		entry.ObservedRequestBodyBytes = &n
	}
}

func applyOpsObservedResponseBody(entry *service.OpsInsertErrorLogInput, w *opsCaptureWriter) {
	if w == nil {
		return
	}
	entry.ObservedResponseBody = w.bodyString()
	entry.ObservedResponseBodyTruncated = w.bodyTruncated()
	if w.Status() >= http.StatusBadRequest {
		n := w.bodyBytesLen()
		entry.ObservedResponseBodyBytes = &n
	}
}
