package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

var serverName string
var zkServer string
var httpPort string

func main() {
	serverName = os.Getenv("SERVER_NAME")
	zkServer = os.Getenv("ZK_SERVER")
	httpPort = os.Getenv("HTTP_PORT")
	if serverName == "" || zkServer == "" || httpPort == "" {
		panic("SERVER_NAME, ZK_SERVER, HTTP_PORT are required")
	}

	createNode()

	listen()
}

func createNode() {
	const Node = "/greet"

	conn, _, _ := zk.Connect([]string{zkServer}, time.Second*5)

	create := func() error {
		var err error
		// try creating ephemeral node
		_, err = conn.Create(Node, []byte(httpPort), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
		return err
	}

	if create() != nil {
		// watch ephemeral node event.
		another, _, eventChan, _ := conn.GetW(Node)
		fmt.Println("Now listen", string(another))
	loop:
		for {
			event := <-eventChan
			if event.Type == zk.EventNodeDeleted || event.Type.String() == "Unknown" {
				// retry creating ephemeral node
				if create() != nil {
					break loop
				}
			}
		}
	}
}

func listen() {
	fmt.Println("Listen", httpPort)
	http.HandleFunc("/", greet)
	http.ListenAndServe(":"+httpPort, nil)
}

func greet(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	fmt.Println("Greeted", name)
	fmt.Fprintf(w, "Hello %s! I'm %s", name, serverName)
}
