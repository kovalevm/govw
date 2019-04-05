Golang Vowpal Wabbit client
===========================

Install
-------

```
$ go get github.com/kovalevm/govw
```

Usage
-----

```go
package main

import (
	"fmt"
	"log"
	
	"github.com/kovalevm/govw"
)

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

	// And then we can send data for prediction
	p, err := client.Predict("1 tag_name| 100:1 200:0.45 250:0.8")
	if err != nil {
		log.Fatal("Predicting error: ", err)
	}

	fmt.Printf("Prediction result: %f | tag: %s\n", p[0].Value, p[0].Tag)
}
```

