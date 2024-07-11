package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/things-go/go-socks5"
)

var listenOn string
var logFile string

func init() {
	flag.StringVar(&listenOn, "listen", ":10800", "the addr that sock5 server listen on ")
	flag.StringVar(&logFile, "log", "", "log file")
}

func main() {
	// Create a SOCKS5 server
	flag.Parse()
	out := os.Stdout
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		out = f
	}

	server := socks5.NewServer(
		socks5.WithLogger(socks5.NewLogger(log.New(out, "socks5: ", log.LstdFlags))),
	)

	// Create SOCKS5 proxy on localhost port 8000
	fmt.Printf("listenOn:%s\n", listenOn)
	if logFile != "" {
		fmt.Printf("logFile:%s\n", logFile)
	}

	if err := server.ListenAndServe("tcp", listenOn); err != nil {
		panic(err)
	}
}
