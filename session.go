package n6bagent

import (
    "bytes"
    "encoding/binary"
    "io"
)

type Session struct {
    SessionID uint32
    connected bool
    buf       []byte
    sendCh    chan<- io.Reader
    readCh    chan []byte
}

func (s *Session) Read(p []byte) (n int, err error) {
    total := len(p)
    current := 0
    for {
        if len(s.buf) > len(p) {
            copy(p, s.buf)
            s.buf = s.buf[len(p):]
            return total, nil
        } else if len(s.buf) == len(p) {
            copy(p, s.buf)
            s.buf = nil
            return total, nil
        } else {
            copy(p, s.buf)
            current += len(s.buf)
            p = p[len(s.buf):]
            var ok bool
            s.buf, ok = <-s.readCh
            if !ok {
                return current, io.EOF
            }
        }
    }
}

func (s *Session) Write(p []byte) (n int, err error) {
    buf := &bytes.Buffer{}
    if !s.connected {
        binary.Write(buf, binary.LittleEndian, 0)
        binary.Write(buf, binary.LittleEndian, len(p)+7)
        binary.Write(buf, binary.LittleEndian, s.SessionID)
        buf.Write([]byte{'B', 'O', 'F'})
        buf.Write(p)
    } else {
        binary.Write(buf, binary.LittleEndian, s.SessionID)
        binary.Write(buf, binary.LittleEndian, len(p))
        buf.Write(p)

    }
    s.sendCh <- buf
    return len(p), nil
}

func (s *Session) Close() {
    buf := &bytes.Buffer{}
    binary.Write(buf, binary.LittleEndian, 0)
    binary.Write(buf, binary.LittleEndian, 7)
    binary.Write(buf, binary.LittleEndian, s.SessionID)
    buf.Write([]byte{'E', 'O', 'F'})
    s.sendCh <- buf
    s.connected = false
}
