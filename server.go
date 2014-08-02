package n6bagent

import (
    "bufio"
    // "bytes"
    "code.google.com/p/go.net/websocket"
    // "encoding/binary"
    "github.com/hashicorp/yamux"
    "io"
    "log"
    "net"
    "net/http"
    "net/url"
    "runtime/pprof"
)

const html = `
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN"
	"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">

<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en" lang="en">
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
	<title>n6bagent</title>
</head>

<body>
    <h1><a href="http://github.com/tiancaiamao/n6bagent">n6bagent</a></h1>

      Welcome to use n6bagent!
</body>
</html>
`

type Server struct {
    *http.ServeMux
}

func NewServer() *Server {
    mux := http.NewServeMux()
    mux.HandleFunc("/", indexCallback)
    mux.Handle("/websocket", websocket.Handler(websocketCallback))
    mux.HandleFunc("/pprof/goroutine", debugCallback("goroutine"))
    mux.HandleFunc("/pprof/heap", debugCallback("heap"))
    mux.HandleFunc("/pprof/threadcreate", debugCallback("threadcreate"))
    mux.HandleFunc("/pprof/block", debugCallback("block"))
    return &Server{mux}
}

// hello world, the web server
func indexCallback(w http.ResponseWriter, req *http.Request) {
    io.WriteString(w, html)
}

func debugCallback(name string) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, req *http.Request) {
        g := pprof.Lookup(name)
        if g != nil {
            g.WriteTo(w, 2)
        } else {
            io.WriteString(w, html)
        }
    }
}

func worker(conn net.Conn) {
    bufreader := bufio.NewReader(conn)
    req, err := http.ReadRequest(bufreader)
    if err != nil {
        log.Println("read request error:", conn.RemoteAddr(), err)
        return
    }
    req.URL, err = url.Parse("http://" + req.Host + req.URL.String())
    req.RequestURI = ""

    log.Printf("SESSION %s BEGIN\n", conn.RemoteAddr())

    if oconn, err := net.DialTimeout("tcp", req.Host, 5); err == nil {
        go io.Copy(conn, oconn)
        io.Copy(oconn, conn)
    }

    // if req, err := http.NewRequest(ireq.Method, ireq.URL.String(), ireq.Body); err == nil {
    //      for k, values := range ireq.Header {
    //          for _, v := range values {
    //              req.Header.Add(k, v)
    //          }
    //      }
    //      req.ContentLength = ireq.ContentLength
    //      // do not follow any redirect， browser will do that
    //      if resp, err := http.DefaultTransport.RoundTrip(req); err == nil {
    //          for k, values := range resp.Header {
    //              for _, v := range values {
    //                  w.Header().Add(k, v)
    //              }
    //          }
    //          defer resp.Body.Close()
    //          w.WriteHeader(resp.StatusCode)
    //          io.Copy(conn, resp.Body)
    //      }
    //  }

    log.Printf("SESSION %d END\n", conn.RemoteAddr())
}

func websocketCallback(ws *websocket.Conn) {
    session, err := yamux.Server(ws, nil)
    if err != nil {
        log.Println("websocket serve error:", err)
        ws.Close()
    }

    for {
        conn, err := session.Accept()
        if err != nil {
            log.Println("session Accept error:", err)
            continue
        }

        go worker(conn)
    }
}

func (s *Server) fetchDirectly(w http.ResponseWriter, ireq *http.Request) {
    if req, err := http.NewRequest(ireq.Method, ireq.URL.String(), ireq.Body); err == nil {
        for k, values := range ireq.Header {
            for _, v := range values {
                req.Header.Add(k, v)
            }
        }
        req.ContentLength = ireq.ContentLength
        // do not follow any redirect， browser will do that
        if resp, err := http.DefaultTransport.RoundTrip(req); err == nil {
            for k, values := range resp.Header {
                for _, v := range values {
                    w.Header().Add(k, v)
                }
            }
            defer resp.Body.Close()
            w.WriteHeader(resp.StatusCode)
            io.Copy(w, resp.Body)
        }
    }
}

// func (s *Server) tunnelTraffic(w http.ResponseWriter, r *http.Request) {
//		 w.WriteHeader(200)
//
//		 if iconn, _, err := w.(http.Hijacker).Hijack(); err == nil {
//				 proxy := s.getProxy()
//				 log.Printf("socks tunnel by %v: %v", proxy.Addr, r.URL.Host)
//
//				 if oconn, err := proxy.Dial(Timeout, r.URL.Host); err == nil {
//						 go copyConn(iconn, oconn)
//						 go copyConn(oconn, iconn)
//				 } else {
//						 log.Println("dial socks server %v, error: %v", proxy.Addr, err)
//						 iconn.Close()
//				 }
//		 } else {
//				 http.Error(w, err.Error(), http.StatusInternalServerError)
//		 }
// }
