package main

import (
	"github.com/tiancaiamao/n6bagent/proxy"
	"log"
	"net"
	"sync/atomic"
)

const (
	MAX_READ_CHUNK_SIZE = 8192
)

var seed uint32 = 0

func handleConn(conn *net.TCPConn, proxyServerType int) {
	sessionID := atomic.AddUint32(&seed, 1)
	proxy.HandleConn(sessionID, conn, proxyServerType)
}

func handleServer(lp *net.TCPListener, proxyServerType int) {
	for {
		conn, err := lp.AcceptTCP()
		if nil != err {
			continue
		}
		go handleConn(conn, proxyServerType)
	}
}

func startLocalProxyServer(addr string, proxyServerType int) bool {
	tcpaddr, err := net.ResolveTCPAddr("tcp", addr)
	if nil != err {
		return false
	}
	var lp *net.TCPListener
	lp, err = net.ListenTCP("tcp", tcpaddr)
	if nil != err {
		log.Fatalf("Can NOT listen on address:%s\n", addr)
		return false
	}
	log.Printf("Listen on address %s\n", addr)
	handleServer(lp, proxyServerType)
	return true
}

func main() {
	go startLocalProxyServer("localhost:48102", proxy.C4_PROXY_SERVER)
	startLocalProxyServer("localhost:48100", proxy.GLOBAL_PROXY_SERVER)
}
