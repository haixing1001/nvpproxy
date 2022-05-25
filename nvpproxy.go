package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
)

func checkError(err error) {
	if err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}

func handleClient(p1, p2 net.Conn) {
	//log.Println("stream opened")
	//defer log.Println("stream closed")
	defer p1.Close()
	defer p2.Close()

	// start tunnel
	p1die := make(chan struct{})
	go func() {
		io.Copy(p1, p2)
		close(p1die)
	}()

	p2die := make(chan struct{})
	go func() {
		io.Copy(p2, p1)
		close(p2die)
	}()

	// wait for tunnel termination
	select {
	case <-p1die:
	case <-p2die:
	}
}

func handleHttpProxy(conn net.Conn, proxy string) (err error) {
	defer conn.Close()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if req == nil || req.Host == "" {
		//log.Println("req is nil or empty, ignore it")
		return
	}

	log.Println("new conn from", conn.RemoteAddr(), "host=", req.Host)

	req.Host = proxy

	remote, err := net.Dial("tcp", proxy)
	if err != nil {
		log.Println("dial vpn error")
		return err
	}
	defer remote.Close()

	if req.Method == "CONNECT" {
		//log.Println("conn, host=", req.Host)
		b := []byte("HTTP/1.1 200 Connection established\r\n" +
			"Proxy-Agent: KcpTun\r\n\r\n")

		if _, err := conn.Write(b); err != nil {
			log.Println("method == CONNECT and err=", err)
			return err
		}
	} else {
		//log.Println("why write, host=", req.Host)
		err = req.Write(remote)
		if err != nil {
			log.Println("method != CONNECT and err=", err)
			return
		}
	}

	//log.Println("CONNECT", req.Host, "OK")
	handleClient(conn, remote)
	return
}

func main() {
	var proxy string
	var port int

	flag.StringVar(&proxy, "proxy", "127.0.0.1:1194", "server of open-v-p-n")
	flag.IntVar(&port, "port", 18888, "local port of proxy")
	flag.Parse()

	addr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(port))
	checkError(err)
	listener, err := net.ListenTCP("tcp", addr)
	checkError(err)
	log.Println("proxy listening on:", listener.Addr())

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Println(err)
			continue
		}

		conn.SetNoDelay(false)
		go handleHttpProxy(conn, proxy)
	}
}
