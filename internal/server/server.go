package server

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync/atomic"

	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
)

type Handler func(w io.Writer, req *request.Request) *HandlerError

type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

type Server struct {
	listener net.Listener
	closed   atomic.Bool
	handler  Handler
}

func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	server := &Server{
		listener: listener,
		closed:   atomic.Bool{},
		handler:  handler,
	}

	go server.listen()

	return server, nil
}

func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}
			continue
		}

		go s.handle(conn)
	}
}

func writeHandlerError(w io.Writer, handlerErr *HandlerError) error {
	body := handlerErr.Message
	headers := response.GetDefaultHeaders(len(body))

	err := response.WriteStatusLine(w, handlerErr.StatusCode)
	if err != nil {
		return err
	}

	err = response.WriteHeaders(w, headers)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(body))
	return err
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	req, err := request.RequestFromReader(conn)
	if err != nil {
		return
	}

	var buf bytes.Buffer
	handlerErr := s.handler(&buf, req)

	if handlerErr != nil {
		writeHandlerError(conn, handlerErr)
		return
	}

	body := buf.Bytes()
	headers := response.GetDefaultHeaders(len(body))

	err = response.WriteStatusLine(conn, response.StatusOK)
	if err != nil {
		return
	}

	err = response.WriteHeaders(conn, headers)
	if err != nil {
		return
	}

	_, err = conn.Write(body)
	if err != nil {
		return
	}
}
