package main

import (
	"bufio"
	"bytes"
	"code.google.com/p/go.net/websocket"
	"encoding/binary"
	"errors"
	// "fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	// "os"
	"sync"
	"sync/atomic"
)

var session uint32

type Cell struct {
	w http.ResponseWriter
	r *http.Request
	c chan struct{}
}
type Result struct {
	sync.Mutex
	kv map[uint32]*Cell
}

func NewResult() *Result {
	return &Result{
		kv: make(map[uint32]*Cell),
	}
}

func (r *Result) Put(session uint32, w *Cell) {
	r.Lock()
	r.kv[session] = w
	r.Unlock()
}

func (r *Result) Del(session uint32) (*Cell, error) {
	r.Lock()
	ret, ok := r.kv[session]
	if !ok {
		r.Unlock()
		return nil, errors.New("not exist")
	}
	r.Unlock()
	return ret, nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	sessionID := atomic.AddUint32(&session, 1)
	log.Printf("SESSION %d BEGIN: %s %s\n", sessionID, r.Method, r.URL.String())

	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, sessionID)
	binary.Write(buf, binary.LittleEndian, uint32(0))
	r.Write(buf)

	b := buf.Bytes()
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b)-8))

	wait := make(chan struct{})
	// 等待返回结束
	Res.Put(sessionID, &Cell{
		w, r, wait,
	})

	Ch <- b
	<-wait
	log.Printf("SESSION %d END\n", sessionID)
}

func sender(ws *websocket.Conn) {
	for {
		buf := <-Ch

		wn := 0
		total := len(buf)
		for wn != total {
			n, err := ws.Write(buf[wn:total])
			if err != nil {
				log.Println("handleConn Write error: ", err)
				return
			}
			wn += n
		}
	}
}

const (
	SESSION = iota
	SIZE
	CONTENT
)

func readSession(ws *websocket.Conn, buf *bytes.Buffer) (uint32, error) {
	var session uint32
	var size uint32

	err := binary.Read(ws, binary.LittleEndian, &session)
	if err != nil {
		return 0, err
	}
	binary.Read(ws, binary.LittleEndian, &size)
	if err != nil {
		return 0, err
	}

	buf.Reset()
	log.Printf("session %d size = %d\n", session, size)
	_, err = io.CopyN(buf, ws, int64(size))
	if err != nil {
		return 0, err
	}
	return session, nil
}

func receiver(ws *websocket.Conn) {
	buf := &bytes.Buffer{}
	for {
		session, err := readSession(ws, buf)
		if err != nil {
			log.Fatal("readSession error: ", err)
		}

		c, err := Res.Del(session)
		if err != nil {
			log.Printf("session %d not exist!!!", session)
			continue
		}

		bufreader := bufio.NewReader(buf)
		resp, err := http.ReadResponse(bufreader, c.r)
		if err != nil {
			log.Printf("read response error!!!!")
			c.c <- struct{}{}
			continue
		}

		for k, v := range resp.Header {
			for _, vv := range v {
				c.w.Header().Add(k, vv)
			}
		}

		result, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("读Body也可以出错?", err)
			c.c <- struct{}{}
			continue
		}

		c.w.Write(result)
		if err != nil {
			log.Println("写回resp错误：", err)
		}
		c.c <- struct{}{}
	}
}

var (
	Ch  = make(chan []byte)
	Res = NewResult()
)

func main() {
	origin := "http://n6bagent-c9-tiancaiamao.c9.io/"
	url := "ws://n6bagent-c9-tiancaiamao.c9.io:80/websocket"

	// origin := "http://localhost/"
	// url := "ws://localhost:8080/websocket"
	
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	go sender(ws)
	go receiver(ws)

	http.HandleFunc("/", Handler)
	http.ListenAndServe(":48101", nil)
}
