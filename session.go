package n6bagent

type Session struct {
    SessionID uint64
    buf       []byte
    sendCh    chan<- io.ReadCloser
    readCh    <-chan []byte
}

func (s *Session) Read(p []byte) (n int, err error) {
    total := len(p)
    current := 0
    for {
        if len(s.buf) > len(p) {
            copy(p, s.buf)
            s.buf = s.buf[len(p):]
            return total, nil
        } else if s.buf == len(p) {
            copy(p, s.buf)
            s.buf = nil
            return total, nil
        } else {
            copy(p, s.buf)
            current += len(s.buf)
            p = p[len(s.buf):]
            s.buf, ok = <-readCh
            if !ok {
                return current, io.EOF
            }
        }
    }
}

func (s *Session) Write(p []byte) (n int, err error) {
    buf := &bytes.Buffer{}
    binary.Write(buf, binary.LittleEndian, sessionID)
    binary.Write(buf, binary.LittleEndian, len(p))
    buf.Write(p)
    s.writeCh <- buf
}

type StreamMultiplex struct {
    seed uint64
    sync.Mutex
    sessions map[uint64]*Session
    sendChan <-chan []byte
}

func NewStreamMultiplex(conn io.ReadWriter) *StreamMultiplex {
    ret = &StreamMultiplex{}
    go ret.backgroundRead(conn)
    go ret.backgroundWrite(conn)
    return ret
}

func (mux *StreamMultiplex) backgroundRead(reader io.Reader) {
    var session uint32
    var size uint32

    err := binary.Read(reader, binary.LittleEndian, &session)
    if err != nil {
        return
    }
    err = binary.Read(reader, binary.LittleEndian, &size)
    if err != nil {
        return
    }

    s := &Session{
        buf: &bytes.Buffer{},
    }
    var ok bool

    mux.Lock()
    s, ok = mux[session]
    mux.Unlock()

    if !ok || s == nil {
        log.Println("backgroundRead error: read a non-exist session")
        io.NewLimitReader(reader, size).ReadAll() // read and drop data
        continue
    }

    log.Printf("session %d size = %d\n", session, size)
    _, err = io.CopyN(session.buf, reader, int64(size))
    if err != nil {
        return
    }
    return session, nil
}

func (mux *StreamMultiplex) backgroundWrite(writer io.Writer) {
    for {
        reader := <-mux.sencChan
        io.Copy(writer, reader)
    }
}

func (mux *StreamMultiplex) New() *Session {
    session := &Session{}
    mux.Lock()
    mux.seed++
    mux[mux.seed] = session
    session.SessionID = mux.seed
    mux.Unlock()

    return session
}

func (mux *StreamMultiplex) Free(sessionId uint64) {
    mux.Lock()
    session, ok := mux.sessions[sessionId]
    if !ok {
        mux.Unlock()
        return
    }
    close(session.readCh)
    delete(mux.sessions, sessionId)
    mux.Unlock()

    return session
}
