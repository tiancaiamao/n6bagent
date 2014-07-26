package main

import (
	"bufio"
	"code.google.com/p/go.net/websocket"
	"encoding/binary"
	"github.com/tiancaiamao/n6bagent/protocol"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Session struct {
	SessionID uint32
	RawReq    *http.Request
	Conn      *net.TCPConn
	Https     bool
}

type SessionManager struct {
	seed uint32
	kv   map[uint32]*Session
	sync.Mutex
}

func NewSessionManager() *SessionManager {
	ret := new(SessionManager)
	ret.kv = make(map[uint32]*Session)
	return ret
}

func (m *SessionManager) NewSession(conn *net.TCPConn) *Session {
	sessionID := atomic.AddUint32(&m.seed, 1)

	bufreader := bufio.NewReader(conn)
	b, err := bufreader.Peek(7)
	if nil != err {
		if err != io.EOF {
			log.Printf("Failed to peek data:%s\n", err.Error())
		}
		conn.Close()
		return nil
	}

	ret := &Session{
		SessionID: sessionID,
		Conn:      conn,
	}

	if strings.EqualFold(string(b), "Connect") {
		ret.Https = true
	}

	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	req, err := http.ReadRequest(bufreader)
	if nil != req {
		req.Header.Del("Proxy-Connection")
	}
	if err != nil {
		conn.Close()
		return nil
	}
	var zero time.Time
	conn.SetReadDeadline(zero)

	ret.RawReq = req

	m.Lock()
	m.kv[sessionID] = ret
	m.Unlock()

	return ret
}

func (m *SessionManager) GetSession(sessionID uint32) *Session {
	var ret *Session
	m.Lock()
	ret = m.kv[sessionID]
	m.Unlock()

	return ret
}

var RequestChannel chan protocol.HttpRequestEvent = make(chan protocol.HttpRequestEvent)
var session uint32

func handleConn(ws *websocket.Conn, conn net.Conn) {
	sessionID := atomic.AddUint32(&session, 1)
	var buf [8192]byte
	for {
		n, err := conn.Read(buf[6:])
		if err != nil {
			log.Println("handleConn Read error: ", err)
			conn.Close()
			return
		}

		// sessionID + size + data
		binary.LittleEndian.PutUint32(buf[:], sessionID)
		binary.LittleEndian.PutUint16(buf[4:], uint16(n))

		total := 6 + n
		wn := 0
		for wn != total {
			n, err := ws.Write(buf[wn:total])
			if err != nil {
				log.Println("handleConn Read error: ", err)
				ws.Close()
				conn.Close()
				return
			}
			wn += n
		}
	}
}

func sender(ws *websocket.Conn) {
	// enc := gob.NewEncoder(ws)
	for {
		ev, ok := <-RequestChannel
		if !ok {

		}
		log.Printf("Session %d: %s %s", ev.SessionID, ev.RawReq.Method, ev.RawReq.URL)
		// err := enc.Encode(ev)
		// ws.Write([]byte("hello, world!\n"))
		err := websocket.Message.Send(ws, ev)
		if err != nil {
			log.Printf("encode error...%v\n", err)
		}
	}
}

func receiver(ws *websocket.Conn) {
	var msg = make([]byte, 512)
	for {
		n, err := ws.Read(msg)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Received: %s.\n", msg[:n])
	}
}

func main() {
	origin := "http://n6bagent-c9-tiancaiamao.c9.io/"
	url := "ws://n6bagent-c9-tiancaiamao.c9.io:80/websocket"

	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	// if _, err := ws.Write([]byte("hello, world!\n")); err != nil {
	//		 log.Fatal(err)
	//	 }
	//	 var msg = make([]byte, 512)
	//	 var n int
	//	 if n, err = ws.Read(msg); err != nil {
	//		 log.Fatal(err)
	//	 }
	//	 log.Printf("Received: %s.\n", msg[:n])

	go sender(ws)
	go receiver(ws)

	fd, err := net.Listen("tcp", "localhost:48101")
	if nil != err {
		log.Fatal(err)
	}

	for {
		conn, err := fd.Accept()
		if nil != err {
			continue
		}
		go handleConn(ws, conn)
	}
}
