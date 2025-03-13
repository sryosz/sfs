package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	sfsencoding "github.com/sryosz/sharing-file-system/internal/encoding"
	tcptransport "github.com/sryosz/sharing-file-system/internal/peers/tcp"
	"github.com/sryosz/sharing-file-system/internal/server"
	"github.com/sryosz/sharing-file-system/internal/storage"
)

func main() {
	s1 := makeServer(":1000")
	s2 := makeServer(":3000")
	s3 := makeServer(":5000", ":1000", ":3000")

	go func() {
		s1.Run()
	}()

	time.Sleep(1 * time.Second)

	go func() {
		s2.Run()
	}()

	time.Sleep(2 * time.Second)

	go s3.Run()

	data := bytes.NewReader([]byte("my big data file"))
	if err := s3.Store("test", data); err != nil{
		log.Fatal(err)
	}

	f, err := s3.Get("test")
	if err != nil{
		panic(err)
	}

	b, err := io.ReadAll(f)
	if err != nil{
		panic(err)
	}

	fmt.Println(string(b))

	select {}
}

func makeServer(listenAddr string, nodes ...string) *server.Server {
	tcpTransportOpts := tcptransport.TCPTransportOpts{
		ListenAddr: listenAddr,
		Decoder:    &sfsencoding.DefaultDecoder{},
	}

	tcpTransport := tcptransport.NewTCPTransport(tcpTransportOpts)

	serverOpts := server.ServerOpts{
		ID:                listenAddr,
		StorageRoot:       strings.ReplaceAll(listenAddr, ":", "") + "_network",
		PathFunc: storage.CASPathFunc,
		Transport:         tcpTransport,
		Nodes:             nodes,
	}

	s := server.NewServer(serverOpts)

	tcpTransport.OnPeer = s.OnPeer

	return s
}
