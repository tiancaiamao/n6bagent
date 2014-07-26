package main

import (
	"bufio"
	"log"
	"net"
	"net/http"
)

func main() {
	fd, err := net.Listen("tcp", ":"+os.Getenv("PORT"))
	if nil != err {
		log.Fatal(err)
	}

	for {
		conn, err := fd.Accept()
		if nil != err {
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	var client http.Client

	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("read request error:", err)
		return
	}
	req.RequestURI = ""

	resp, err := client.Do(req)
	if err != nil {
		log.Println("client Do error:", err)
		return
	}

	resp.Write(conn)
}
