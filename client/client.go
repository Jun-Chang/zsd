package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"text/scanner"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

const Node = "/greet"

var zkServer string
var zkConn *zk.Conn

var servicePort string
var servicePortMtx sync.Mutex

type NoLog struct{}

func (NoLog) Printf(string, ...interface{}) {}

func main() {
	zkServer = os.Getenv("ZK_SERVER")
	if zkServer == "" {
		panic("ZK_SERVER is required")
	}
	zkConn, _, _ = zk.Connect([]string{zkServer}, time.Second*5)
	zkConn.SetLogger(new(NoLog))

	go watch()

	var s scanner.Scanner
	s.Init(os.Stdin)
	for {
		fmt.Print("Enter your name.\n> ")
		s.Scan()
		fmt.Println(call(s.TokenText()))
	}
}

func call(name string) string {
	var p string
	for {
		if p = discover(false); p != "" {
			break
		}
	}
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/?name=%s", p, name))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, _ := ioutil.ReadAll(resp.Body)

	return string(b)
}

func discover(change bool) string {
	servicePortMtx.Lock()
	defer servicePortMtx.Unlock()

	if change || servicePort == "" {
		p, _, _ := zkConn.Get(Node)
		servicePort = string(p)
	}

	return servicePort
}

func watch() {
	var eventChan <-chan zk.Event
	_, _, eventChan, _ = zkConn.GetW(Node)
	for {
		event := <-eventChan
		if event.Type == zk.EventNodeDeleted || event.Type.String() == "Unknown" {
			discover(true)
		}
		_, _, eventChan, _ = zkConn.GetW(Node)
	}
}
