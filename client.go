package n6bagent

import (
    "bufio"
    "bytes"
    "code.google.com/p/go.net/websocket"
    "encoding/binary"
    "errors"
    "github.com/hashicorp/yamux"
    // "fmt"
    "io"
    "io/ioutil"
    "log"
    "net/http"
    // "os"
    "sync"
)

type Cell struct {
    w   http.ResponseWriter
    r   *http.Request
    c   chan struct{}
}
type Result struct {
    sync.Mutex
    kv  map[uint32]*Cell
}

func newResult() *Result {
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

func sender(ws *websocket.Conn, ch <-chan []byte) {
    for {
        buf := <-ch

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

func receiver(ws *websocket.Conn, res *Result) {
    buf := &bytes.Buffer{}
    for {
        session, err := readSession(ws, buf)
        if err != nil {
            log.Fatal("readSession error: ", err)
        }

        c, err := res.Del(session)
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

type Client struct {
    multiplex *yamux.Session
}

func NewClient(hostAddr string) (*Client, error) {
    origin := "http://" + hostAddr + "/"
    url := "ws://" + hostAddr + "/websocket"

    ws, err := websocket.Dial(url, "", origin)
    if err != nil {
        return nil, err
    }

    ret := new(Client)
    ret.multiplex, err = yamux.Client(ws, nil)
    return ret, err
}

func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    conn, err := c.multiplex.Open()
    if err != nil {
        // 写500内部错误
        return
    }

    log.Printf("SESSION %d BEGIN: %s %s\n", conn.LocalAddr(), r.Method, r.URL.String())

    w.WriteHeader(200)

    if iconn, _, err := w.(http.Hijacker).Hijack(); err == nil {
        go io.Copy(iconn, conn)
        io.Copy(conn, iconn)
    } else {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }

    log.Printf("SESSION %d END\n", conn.LocalAddr())
}

// func (s *Server) tunnelTraffic(w http.ResponseWriter, r *http.Request) {
//     w.WriteHeader(200)
//
//     if iconn, _, err := w.(http.Hijacker).Hijack(); err == nil {
//         proxy := s.getProxy()
//         log.Printf("socks tunnel by %v: %v", proxy.Addr, r.URL.Host)
//
//         if oconn, err := proxy.Dial(Timeout, r.URL.Host); err == nil {
//             go copyConn(iconn, oconn)
//             go copyConn(oconn, iconn)
//         } else {
//             log.Println("dial socks server %v, error: %v", proxy.Addr, err)
//             iconn.Close()
//         }
//     } else {
//         http.Error(w, err.Error(), http.StatusInternalServerError)
//     }
// }
