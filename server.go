package main

import (
    "code.google.com/p/go.net/websocket"
    "github.com/elazarl/goproxy"
    "github.com/hashicorp/yamux"
    "io"
    "log"
    "net/http"
    _ "net/http/pprof"
    "os"
)

func indexCallback(w http.ResponseWriter, req *http.Request) {
    io.WriteString(w, html)
}

func websocketCallback(ws *websocket.Conn) {
    session, err := yamux.Server(ws, nil)
    if err != nil {
        log.Println("websocket serve error:", err)
        ws.Close()
        return
    }

    proxy := goproxy.NewProxyHttpServer()
    // proxy.Verbose = true
    err = http.Serve(session, proxy)
    if err != nil {
        log.Println("Serve error:", err)
        ws.Close()
        return
    }
}

func main() {
    var port = os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    http.HandleFunc("/", indexCallback)
    http.Handle("/websocket", websocket.Handler(websocketCallback))
    http.ListenAndServe(":"+port, nil)
}

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
