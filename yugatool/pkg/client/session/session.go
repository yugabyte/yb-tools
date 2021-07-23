package session

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/config"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/dial"
)

type Session struct {
	m            sync.Mutex
	conn         io.ReadWriteCloser
	messageCount int32

	Dialer dial.Dialer
	Ping   func(*Session) error
}

func NewSession(host string, universeConfig *config.UniverseConfig, ping func(*Session) error) (*Session, error) {
	dialer, err := universeConfig.GetDialer()
	if err != nil {
		return &Session{}, err
	}

	s := &Session{
		Dialer: dialer,
		Ping:   ping,
	}
	err = s.Connect(host)
	return s, err
}

func (s *Session) Lock() {
	s.m.Lock()
}

func (s *Session) Unlock() {
	s.m.Unlock()
}

func (s *Session) Write(bytes []byte) (int, error) {
	return s.conn.Write(bytes)
}

func (s *Session) Read(b []byte) (int, error) {
	return s.conn.Read(b)
}

func (s *Session) Close() error {
	return s.conn.Close()
}

func (s *Session) GenerateCallID() int32 {
	count := atomic.AddInt32(&s.messageCount, 1)
	return count
}

func (s *Session) Connect(host string) error {
	var err error
	s.conn, err = s.Dialer.Dial("tcp", host)
	if err != nil {
		return err
	}

	err = writeHello(s.conn)
	if err != nil {
		_ = s.conn.Close()
		return err
	}

	err = s.Ping(s)
	if err != nil {
		_ = s.conn.Close()
		return err
	}

	return nil
}

func writeHello(conn io.ReadWriteCloser) error {
	var b bytes.Buffer
	b.WriteRune('Y')
	b.WriteRune('B')
	b.WriteByte('\001')
	n, err := conn.Write(b.Bytes())
	if err != nil {
		return err
	}

	if n != 3 {
		return errors.New("hello did not write 3 bytes")
	}
	return nil
}
