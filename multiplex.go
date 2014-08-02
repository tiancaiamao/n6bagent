package n6bagent

import (
    "encoding/binary"
    "io"
    "io/ioutil"
    "log"
    "sync"
)

// 协议：
// 每条消息都是 SessionID | Size | Data
// SessionID = 0 保留作为控制协议
// 连接请求的Data是 SessionID + 'BOF' + 捎带数据
// 连续关闭的Data是 SessionID + 'EOF'

type StreamMultiplex struct {
    seed uint32
    sync.Mutex
    sessions map[uint32]*Session
    sendChan <-chan io.ReadCloser
    listen   bool
    accept   chan *Session
}

func NewStreamMultiplex(conn io.ReadWriter) *StreamMultiplex {
    ret := &StreamMultiplex{}
    go ret.backgroundRead(conn)
    go ret.backgroundWrite(conn)
    return ret
}

// 必须调用Listen之后才能调用Accept
func (mux *StreamMultiplex) Accept() *Session {
    session := <-mux.accept
    mux.Lock()
    mux.sessions[session.SessionID] = session
    mux.Unlock()
    return session
}

func (mux *StreamMultiplex) readCommand(reader io.Reader, size uint32) {
    var sessionID uint32
    err := binary.Read(reader, binary.LittleEndian, &sessionID)
    if err != nil {

    }

    var cmd [3]byte
    _, err = reader.Read(cmd[:])
    if err != nil {

    }

    switch string(cmd[:]) {
    case "BOF":
        data := make([]byte, size-7)
        _, err = reader.Read(data)
        if err != nil {

        }
        session := &Session{
            SessionID: sessionID,
            buf:       data,
        }

        mux.accept <- session
    case "EOF":
        delete(mux.sessions, sessionID)
    default:
    }

    if err != nil {

    }
}

func (mux *StreamMultiplex) backgroundRead(reader io.Reader) {
    var sessionID uint32
    var size uint32

    err := binary.Read(reader, binary.LittleEndian, &sessionID)
    if err != nil {
        return
    }
    err = binary.Read(reader, binary.LittleEndian, &size)
    if err != nil {
        return
    }

    if sessionID == 0 {
        mux.readCommand(reader, size)
    } else {
        s := &Session{}
        var ok bool

        mux.Lock()
        s, ok = mux.sessions[sessionID]
        mux.Unlock()

        if !ok || s == nil {
            log.Println("backgroundRead error: read a non-exist session")
            ioutil.ReadAll(io.LimitReader(reader, int64(size))) // read and drop data
            continue
        }

        data := make([]byte, size)
        reader.Read(data)
        s.readCh <- data
    }
}

func (mux *StreamMultiplex) backgroundWrite(writer io.Writer) {
    for {
        reader := <-mux.sendChan
        io.Copy(writer, reader)
    }
}

func (mux *StreamMultiplex) Dial() *Session {
    session := &Session{}
    mux.Lock()
    mux.seed++
    mux.sessions[mux.seed] = session
    session.SessionID = mux.seed
    mux.Unlock()

    return session
}

func (mux *StreamMultiplex) New() *Session {
    session := &Session{}
    mux.Lock()
    mux.seed++
    mux.sessions[mux.seed] = session
    session.SessionID = mux.seed
    mux.Unlock()

    return session
}

func (mux *StreamMultiplex) Free(sessionId uint32) {
    mux.Lock()
    session, ok := mux.sessions[sessionId]
    if !ok {
        mux.Unlock()
        return
    }
    close(session.readCh)
    delete(mux.sessions, sessionId)
    session.Close()
    mux.Unlock()
}
