package main

import (
    "code.google.com/p/go.net/websocket"
    "github.com/hashicorp/yamux"
    "io"
    "log"
    "net"
)

func NewProxy(hostAddr string) (*yamux.Session, error) {
    origin := "http://" + hostAddr + "/"
    url := "ws://" + hostAddr + "/websocket"

    ws, err := websocket.Dial(url, "", origin)
    if err != nil {
        return nil, err
    }

    return yamux.Client(ws, nil)
}

func Handle(c *yamux.Session, lconn net.Conn) {
    rconn, err := c.Open()
    if err != nil {
        log.Println("multiplex打开连接失败...")
        lconn.Close()
        return
    }

    go io.Copy(rconn, lconn)
    io.Copy(lconn, rconn)
}

func main() {
    proxy, err := NewProxy("n6bagent-c9-tiancaiamao.c9.io:80")
    // proxy, err := NewProxy("localhost:8080")
    if err != nil {
        panic(err)
    }

    listener, err := net.Listen("tcp", ":48101")
    if err != nil {
        panic(err)
    }
    for {
        if conn, err := listener.Accept(); err == nil {
            go Handle(proxy, conn)
        }
    }

}
