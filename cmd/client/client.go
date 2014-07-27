package main

import (
	// "bufio"
	"bytes"
	"code.google.com/p/go.net/websocket"
	"encoding/binary"
	"errors"
	"io"
	// "io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
)

var session uint32

type Cell struct {
	w http.ResponseWriter
	r *http.Request
}
type Result struct {
	sync.Mutex
	kv map[uint32]Cell
}

func NewResult() *Result {
	return &Result{
		kv: make(map[uint32]Cell),
	}
}

func (r *Result) Put(session uint32, w Cell) {
	r.Lock()
	r.kv[session] = w
	r.Unlock()
}

func (r *Result) Del(session uint32) (Cell, error) {
	r.Lock()
	ret, ok := r.kv[session]
	if !ok {
		r.Unlock()
		return Cell{}, errors.New("not exist")
	}
	r.Unlock()
	return ret, nil
}

func NewHandler(ch chan<- []byte, finish *Result) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("get a request: ")
		r.Write(os.Stdout)
		sessionID := atomic.AddUint32(&session, 1)

		buf := &bytes.Buffer{}
		binary.Write(buf, binary.LittleEndian, sessionID)
		binary.Write(buf, binary.LittleEndian, uint16(0))
		r.Write(buf)

		b := buf.Bytes()
		binary.LittleEndian.PutUint16(b[4:], uint16(len(b)-6))
		finish.Put(sessionID, Cell{w, r})
		ch <- b
	}
}

func sender(ws *websocket.Conn, c <-chan []byte) {
	for {
		buf := <-c

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

func receiver(ws *websocket.Conn, res *Result) {
	buf := &bytes.Buffer{}

	var session uint32
	var size uint16

	for {
		_, err := io.CopyN(buf, ws, 4)
		if err != nil {
			log.Fatal(err)
		}
		binary.Read(buf, binary.LittleEndian, &session)
		buf.Reset()

		_, err = io.CopyN(buf, ws, 2)
		if err != nil {
			log.Fatal(err)
		}
		binary.Read(buf, binary.LittleEndian, &size)
		buf.Reset()

		_, err = io.CopyN(buf, ws, int64(size))
		c, err := res.Del(session)
		if err == nil {
			// bufreader := bufio.NewReader(buf)
			// resp, err := http.ReadResponse(bufreader, c.r)
			// if err == nil {
			// 	log.Println("get a response:")
			// 	resp.Write(os.Stdout)
			//
			// 	for k, v := range resp.Header {
			// 		for _, vv := range v {
			// 			c.w.Header().Add(k, vv)
			// 		}
			// 	}
			//
			// 	// c.w.WriteHeader(resp.StatusCode)
			// 	result, err := ioutil.ReadAll(resp.Body)
			// 	if err != nil && err != io.EOF {
			// 		log.Println("是否是运行到这里?", err)
			// 	}
			// 	c.w.Write(result)
			// } else {
			// 	log.Printf("read response error!!!!")
			// }

			_, err = io.Copy(c.w, buf)
			if err != nil {
				log.Println("send to browser error:", err)
			}
		} else {
			log.Printf("session not exist!!!")
		}
		buf.Reset()
	}
}

func main() {
	// origin := "http://n6bagent-c9-tiancaiamao.c9.io/"
	// url := "ws://n6bagent-c9-tiancaiamao.c9.io:80/websocket"

	origin := "http://localhost/"
	url := "ws://localhost:8080/websocket"

	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan []byte)
	res := NewResult()
	go sender(ws, ch)
	go receiver(ws, res)

	http.HandleFunc("/", NewHandler(ch, res))
	http.ListenAndServe(":48101", nil)
}
