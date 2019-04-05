package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kovalevm/govw"
)

var daemon *govw.VWDaemon
var client *govw.VWClient

func predictHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error while reading request body:", err)
		}

		p, err := client.Predict(string(body))
		if err != nil {
			log.Fatal("Error while getting predict:", err)
		}

		res, err := json.Marshal(p)
		if err != nil {
			log.Fatal("Error while marshaling prediction result:", err)
		}

		fmt.Fprintf(w, "%s", res)
	default:
		w.WriteHeader(404)
		fmt.Fprint(w, "Method unavailable!")
	}
}

func runServer() *http.Server {
	server := &http.Server{Addr: ":8080"}

	log.Println("Server running on port", server.Addr)

	http.HandleFunc("/", predictHandler)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	return server
}

func main() {
	var err error
	treats := 10
	ports := [2]int{26542, 26543}
	modelPath := "/full/path/to/some.model"
	testOnly := true

	// initialize and run a daemon
	daemon, err = govw.NewDaemon("daemon", ports, treats, modelPath, testOnly, "")
	if err != nil {
		log.Fatal("Error while initializing VW daemon entity!", err)
	}

	if err = daemon.Run(); err != nil {
		log.Fatal("Error while running VW daemon!", err)
	}

	// create a client
	client = govw.NewClient()
	if err = client.Connect("", ports[0], treats/5); err != nil {
		log.Fatal("Error while connecting VW daemon!", err)
	}

	// auto dump the model
	govw.AutoDump(client, modelPath, 30*time.Second)

	runServer()

	// graceful shut down
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down the service...")

	log.Println("Dump the model")
	if err := client.DumpModel(modelPath); err != nil {
		log.Println("Error while dumping the model server!", err)
	}

	log.Println("Disconnect vw daemon")
	client.Disconnect()

	// wait a bit before stop daemon
	time.Sleep(2 * time.Second)

	log.Println("Stop the daemon")
	if err := daemon.Stop(); err != nil {
		log.Fatal("Error while stopped VW daemon!", err)
	}

	log.Println("Service stopped!")
}
