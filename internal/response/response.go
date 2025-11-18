package response

import (
	"errors"
	"fmt"
	"io"

	"httpfromtcp/internal/headers"
)

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

type writerState int

const (
	writerStateInitial writerState = iota
	writerStateStatusLineWritten
	writerStateHeadersWritten
	writerStateBodyWritten
	writerStateChunkedBodyDone
)

type Writer struct {
	w     io.Writer
	state writerState
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:     w,
		state: writerStateInitial,
	}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.state != writerStateInitial {
		return errors.New("WriteStatusLine must be called first")
	}

	var reasonPhrase string
	switch statusCode {
	case StatusOK:
		reasonPhrase = "OK"
	case StatusBadRequest:
		reasonPhrase = "Bad Request"
	case StatusInternalServerError:
		reasonPhrase = "Internal Server Error"
	default:
		reasonPhrase = ""
	}

	var err error
	if reasonPhrase != "" {
		_, err = fmt.Fprintf(w.w, "HTTP/1.1 %d %s\r\n", statusCode, reasonPhrase)
	} else {
		_, err = fmt.Fprintf(w.w, "HTTP/1.1 %d\r\n", statusCode)
	}

	if err != nil {
		return err
	}

	w.state = writerStateStatusLineWritten
	return nil
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
	if w.state != writerStateStatusLineWritten {
		return errors.New("WriteHeaders must be called after WriteStatusLine")
	}

	for key, value := range headers {
		_, err := fmt.Fprintf(w.w, "%s: %s\r\n", key, value)
		if err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w.w, "\r\n")
	if err != nil {
		return err
	}

	w.state = writerStateHeadersWritten
	return nil
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.state != writerStateHeadersWritten {
		return 0, errors.New("WriteBody must be called after WriteHeaders")
	}

	n, err := w.w.Write(p)
	if err != nil {
		return n, err
	}

	w.state = writerStateBodyWritten
	return n, nil
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	if w.state != writerStateHeadersWritten {
		return 0, errors.New("WriteChunkedBody must be called after WriteHeaders")
	}

	if len(p) == 0 {
		return 0, nil
	}

	// Write chunk size in hex, followed by \r\n
	_, err := fmt.Fprintf(w.w, "%x\r\n", len(p))
	if err != nil {
		return 0, err
	}

	// Write chunk data
	n, err := w.w.Write(p)
	if err != nil {
		return n, err
	}

	// Write \r\n after chunk data
	_, err = fmt.Fprintf(w.w, "\r\n")
	if err != nil {
		return n, err
	}

	return n, nil
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	if w.state != writerStateHeadersWritten {
		return 0, errors.New("WriteChunkedBodyDone must be called after WriteHeaders")
	}

	// Write final chunk marker: 0\r\n
	// Trailers will be written after this if needed
	_, err := fmt.Fprintf(w.w, "0\r\n")
	if err != nil {
		return 0, err
	}

	w.state = writerStateChunkedBodyDone
	return 0, nil
}

func (w *Writer) WriteTrailers(h headers.Headers) error {
	if w.state != writerStateChunkedBodyDone {
		return errors.New("WriteTrailers must be called after WriteChunkedBodyDone")
	}

	// Write trailers (formatted just like headers)
	for key, value := range h {
		_, err := fmt.Fprintf(w.w, "%s: %s\r\n", key, value)
		if err != nil {
			return err
		}
	}

	// Write final CRLF after trailers
	_, err := fmt.Fprintf(w.w, "\r\n")
	if err != nil {
		return err
	}

	w.state = writerStateBodyWritten
	return nil
}

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	var reasonPhrase string
	switch statusCode {
	case StatusOK:
		reasonPhrase = "OK"
	case StatusBadRequest:
		reasonPhrase = "Bad Request"
	case StatusInternalServerError:
		reasonPhrase = "Internal Server Error"
	default:
		reasonPhrase = ""
	}

	if reasonPhrase != "" {
		_, err := fmt.Fprintf(w, "HTTP/1.1 %d %s\r\n", statusCode, reasonPhrase)
		return err
	}
	_, err := fmt.Fprintf(w, "HTTP/1.1 %d\r\n", statusCode)
	return err
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()
	h.Set("Content-Length", fmt.Sprintf("%d", contentLen))
	h.Set("Connection", "close")
	h.Set("Content-Type", "text/plain")
	return h
}

func WriteHeaders(w io.Writer, headers headers.Headers) error {
	for key, value := range headers {
		_, err := fmt.Fprintf(w, "%s: %s\r\n", key, value)
		if err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "\r\n")
	return err
}
