package proxy

import (
	"bufio"
	"fmt"
	"github.com/tiancaiamao/n6bagent/event"
	"github.com/tiancaiamao/n6bagent/socks"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	HTTP_TUNNEL  = 1
	HTTPS_TUNNEL = 2
	SOCKS_TUNNEL = 3

	STATE_RECV_HTTP       = 1
	STATE_RECV_HTTP_CHUNK = 2
	STATE_RECV_TCP        = 3
	STATE_SESSION_CLOSE   = 4

	GLOBAL_PROXY_SERVER = 1
	GAE_PROXY_SERVER    = 2
	C4_PROXY_SERVER     = 3
	SSH_PROXY_SERVER    = 4

	GAE_NAME                 = "GAE"
	C4_NAME                  = "C4"
	GOOGLE_NAME              = "Google"
	GOOGLE_HTTP_NAME         = "GoogleHttp"
	GOOGLE_HTTPS_NAME        = "GoogleHttps"
	GOOGLE_HTTPS_DIRECT_NAME = "GoogleHttpsDirect"
	FORWARD_NAME             = "Forward"
	SSH_NAME                 = "SSH"
	AUTO_NAME                = "Auto"
	DIRECT_NAME              = "Direct"
	DEFAULT_NAME             = "Default"

	ATTR_REDIRECT_HTTPS = "RedirectHttps"
	ATTR_CRLF_INJECT    = "CRLF"
	ATTR_DIRECT         = "Direct"
	ATTR_TUNNEL         = "Tunnel"
	ATTR_RANGE          = "Range"
	ATTR_SYS_DNS        = "SysDNS"
	ATTR_PREFER_HOSTS   = "PreferHosts"
	ATTR_APP            = "App"

	MODE_HTTP    = "http"
	MODE_HTTPS   = "httpS"
	MODE_RSOCKET = "rsocket"
	MODE_XMPP    = "xmpp"
)

var total_proxy_conn_num uint32

type RemoteConnection interface {
	Request(conn *SessionConnection, ev event.Event) (err error, res event.Event)
	GetConnectionManager() RemoteConnectionManager
	Close() error
}

type RemoteConnectionManager interface {
	GetRemoteConnection(ev event.Event, attrs map[string]string) (RemoteConnection, error)
	RecycleRemoteConnection(conn RemoteConnection)
	GetName() string
}

type ForwardSocksDialer struct {
	proxyPort int
}

func (f *ForwardSocksDialer) DialTCP(n string, laddr *net.TCPAddr, raddr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", f.proxyPort))
	if nil == err {
		_, port, er := net.SplitHostPort(raddr)
		if nil == er && port != "80" {
			req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Connection: Keep-Alive\r\n\r\n", raddr, raddr)
			conn.Write([]byte(req))
			tmp := make([]byte, 1024)
			conn.Read(tmp)
		}
	}
	return conn, err
}


