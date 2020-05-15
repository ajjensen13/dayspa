package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	addrFlag          = `addr`
	addrFlagShortHand = `a`
	addrDefault       = `:http`
	addrDescription   = `network address to listen for connections on`

	idleTimeoutFlag        = `idle-timeout`
	idleTimeoutDefault     = time.Second * 60
	idleTimeoutDescription = `idle connections timeout`

	readTimeoutFlag        = `read-timeout`
	readTimeoutDefault     = time.Second * 30
	readTimeoutDescription = `read request timeout`

	readHeaderTimeoutFlag        = `read-header-timeout`
	readHeaderTimeoutDefault     = time.Second * 5
	readHeaderTimeoutDescription = `read header request timeout`

	writeTimeoutFlag        = `write-timeout`
	writeTimeoutDefault     = time.Second * 30
	writeTimeoutDescription = `write response timeout`

	maxHeaderBytesFlag        = `max-header-bytes`
	maxHeaderBytesDefault     = http.DefaultMaxHeaderBytes
	maxHeaderBytesDescription = `maximum header bytes to allow in a request`
)

func Init(cmd *cobra.Command) {
	cmd.Flags().StringP(addrFlag, addrFlagShortHand, addrDefault, addrDescription)
	cmd.Flags().Duration(idleTimeoutFlag, idleTimeoutDefault, idleTimeoutDescription)
	cmd.Flags().Duration(readTimeoutFlag, readTimeoutDefault, readTimeoutDescription)
	cmd.Flags().Duration(readHeaderTimeoutFlag, readHeaderTimeoutDefault, readHeaderTimeoutDescription)
	cmd.Flags().Duration(writeTimeoutFlag, writeTimeoutDefault, writeTimeoutDescription)
	cmd.Flags().Int(maxHeaderBytesFlag, maxHeaderBytesDefault, maxHeaderBytesDescription)
}

func RunE(cmd *cobra.Command, _ []string) error {
	addr, err := cmd.Flags().GetString(addrFlag)
	if err != nil {
		return fmt.Errorf("server: failed to get %s flag: %w", addrFlag, err)
	}

	readTimeout, err := cmd.Flags().GetDuration(readTimeoutFlag)
	if err != nil {
		return fmt.Errorf("server: failed to get %s flag: %w", readTimeoutFlag, err)
	}

	readHeaderTimeout, err := cmd.Flags().GetDuration(readHeaderTimeoutFlag)
	if err != nil {
		return fmt.Errorf("server: failed to get %s flag: %w", readHeaderTimeoutFlag, err)
	}

	writeTimeout, err := cmd.Flags().GetDuration(writeTimeoutFlag)
	if err != nil {
		return fmt.Errorf("server: failed to get %s flag: %w", writeTimeoutFlag, err)
	}

	idleTimeout, err := cmd.Flags().GetDuration(idleTimeoutFlag)
	if err != nil {
		return fmt.Errorf("server: failed to get %s flag: %w", idleTimeoutFlag, err)
	}

	maxHeaderBytes, err := cmd.Flags().GetInt(maxHeaderBytesFlag)
	if err != nil {
		return fmt.Errorf("server: failed to get %s flag: %w", maxHeaderBytesFlag, err)
	}

	srv := newServer(addr, readTimeout, readHeaderTimeout, writeTimeout, idleTimeout, maxHeaderBytes)
	return srv.server.ListenAndServe()
}

func newServer(addr string, readTimeout time.Duration, readHeaderTimeout time.Duration, writeTimeout time.Duration, idleTimeout time.Duration, maxHeaderBytes int) *server {
	var srv server
	srv = server{
		server: http.Server{
			Addr:              addr,
			Handler:           http.DefaultServeMux,
			ReadTimeout:       readTimeout,
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			MaxHeaderBytes:    maxHeaderBytes,
			ConnState:         srv.connState,
			ConnContext:       srv.connContext,
		},
		connToKey: map[net.Conn]uuid.UUID{},
		keyToData: map[uuid.UUID]*connData{},
	}
	return &srv
}

type server struct {
	server http.Server

	lock      sync.RWMutex
	connToKey map[net.Conn]uuid.UUID
	keyToData map[uuid.UUID]*connData
}

func (s *server) connState(c net.Conn, state http.ConnState) {
	k, d := s.connData(c)

	d.lock.Lock()
	defer d.lock.Unlock()

	now := time.Now()
	if state == http.StateNew {
		d.begin = now
	}

	d.log = append(d.log, connStateTx{state, now})

	if state == http.StateClosed || state == http.StateHijacked {
		d.end = now
		s.deleteConnData(k, c)
	}
}

type contextKey string

const serverConnContextKey = contextKey(`server-conn`)

func (s *server) connContext(ctx context.Context, c net.Conn) context.Context {
	k := s.newConnData(c)
	return context.WithValue(ctx, serverConnContextKey, k)
}

func (s *server) connData(c net.Conn) (uuid.UUID, *connData) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	k, ok := s.connToKey[c]
	if !ok {
		panic(errors.New("server: connection doesn't exists"))
	}

	d, ok := s.keyToData[k]
	if !ok {
		panic(fmt.Errorf("server: context key %v doesn't exists", k))
	}

	return k, d
}

func (s *server) newConnData(c net.Conn) uuid.UUID {
	s.lock.Lock()
	defer s.lock.Unlock()

	k := uuid.New()
	if _, ok := s.keyToData[k]; ok {
		panic(fmt.Errorf("server: context key %v already exists", k))
	}

	if _, ok := s.connToKey[c]; ok {
		panic(errors.New("server: connection already exists"))
	}

	d := connData{}
	s.keyToData[k] = &d
	s.connToKey[c] = k
	return k
}

func (s *server) deleteConnData(k uuid.UUID, c net.Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.keyToData[k]; !ok {
		panic(fmt.Errorf("server: context key %v doesn't exists", k))
	}

	if _, ok := s.connToKey[c]; !ok {
		panic(errors.New("server: connection doesn't exists"))
	}

	delete(s.keyToData, k)
	delete(s.connToKey, c)
}

type connData struct {
	lock  sync.Mutex
	begin time.Time
	end   time.Time
	c     net.Conn
	log   []tx
}

func (c *connData) String() string {
	result, err := c.MarshalText()
	if err != nil {
		panic(err)
	}
	return string(result)
}

func (c *connData) MarshalText() (text []byte, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var buf bytes.Buffer

	if c.begin.IsZero() {
		return buf.Bytes(), nil
	}

	_, err = fmt.Fprintf(&buf, "BEGIN CONN DATA: %v\n", c.begin)
	if err != nil {
		return nil, err
	}

	for _, t := range c.log {
		_, err = fmt.Fprintf(&buf, " %+06.3fs: ", float64(t.Timestamp().Sub(c.begin).Milliseconds())/1000)
		if err != nil {
			return nil, err
		}

		_, err = t.WriteTo(&buf)
		if err != nil {
			return nil, err
		}
	}

	if c.end.IsZero() {
		_, err = fmt.Fprint(&buf, "...\n")
	} else {
		_, err = fmt.Fprintf(&buf, "END CONN DATA: %v\n", c.end)
	}

	return buf.Bytes(), nil
}

type tx interface {
	io.WriterTo
	Timestamp() time.Time
}

type connStateTx struct {
	state http.ConnState
	ts    time.Time
}

func (c connStateTx) WriteTo(w io.Writer) (n int64, err error) {
	i, err := fmt.Fprintf(w, "CONN STATE = %v\n", c.state)
	return int64(i), err
}

func (c connStateTx) Timestamp() time.Time {
	return c.ts
}
