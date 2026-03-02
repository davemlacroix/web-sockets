package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

func main() {
	fmt.Println("Starting websockets client...")
	//get test cases count by connecting to:
	//ws://localhost:9001/getCaseCount

	conn, err := net.Dial("tcp", "127.0.0.1:9001")
	if err != nil {
		fmt.Println(err)
		fmt.Println("error with initial connection")
		log.Fatal(err)
	}
	defer conn.Close()

	req, err := http.NewRequest("GET", "http://127.0.0.1:9001/getCaseCount", nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Upgrade", "websocket")
	req.Header.Add("Connection", "upgrade")
	req.Header.Add("Upgrade", "websocket")
	req.Header.Add("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Add("Origin", "127.0.0.1")
	req.Header.Add("Sec-WebSocket-Protocol", "chat, superchat")
	req.Header.Add("Sec-WebSocket-Version", "13")

	err = req.Write(conn)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	buf := make([]byte, 1024)
	io.ReadFull(reader, buf)
	fmt.Println(buf)
}
