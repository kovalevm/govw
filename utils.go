package govw

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func runCommand(command string, quiet bool) ([]byte, error) {
	if quiet {
		err := exec.Command("sh", "-c", command).Start()
		if err != nil {
			return []byte{}, err
		}
		return []byte{}, nil
	}

	val, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		return []byte{}, err
	}
	return val, nil
}

// ParsePredictResult get prediction result from VW daemon
// and convert this result into Prediction struct.
func ParsePredictResult(predict *string) (*Prediction, error) {
	p := strings.TrimRight(*predict, "\n")

	r := strings.Split(p, " ")

	val, err := strconv.ParseFloat(r[0], 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing prediction value: %s", err)
	}

	if len(r) == 1 {
		return &Prediction{val, ""}, nil
	}

	return &Prediction{val, r[1]}, nil
}

// RecreateDaemon create new VW daemon on another port (default VW port + 1),
// check if all his children is wakeup, substitute link to new VW daemon instance.
func RecreateDaemon(d *VWDaemon) {
	log.Println("Start recreating daemon on new port:", d.Port[1])

	port := [2]int{d.Port[1], d.Port[0]}
	newVW, err := NewDaemon(d.BinPath, port, d.Children, d.Model.Path, d.Test, d.VwOpts)
	if err != nil {
		log.Fatal("Error while initializing VW daemon entity!", err)
	}

	if err := newVW.Run(); err != nil {
		log.Fatal(err)
	}

	tmpVW := *d
	d = newVW

	tmpVW.Stop()
	log.Println("Finished recreating daemon on new port:", d.Port[0])
}

// ModelFileChecker check if our model file is changed,
// and recreate VW daemon on a new port.
func ModelFileChecker(vw *VWDaemon) {
	for {
		time.Sleep(time.Second * 5) // TODO: Move count of second to settings struct

		isChanged, err := vw.Model.IsChanged()
		if err != nil {
			continue
		}

		if isChanged {
			log.Println("Model file is changed!")
			RecreateDaemon(vw)
		}
	}
}

func CreateTCPConn(host string, port int) (net.Conn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, fmt.Errorf("error resolving IP addr: %s", err)
	}

	conn, err := fasthttp.Dial(tcpAddr.String())
	if err != nil {
		return nil, fmt.Errorf("error dialing TCP: %s", err)
	}

	return conn, nil
}

func AutoDump(c *VWClient, path string, dur time.Duration) {
	ticker := time.NewTicker(dur)
	go func() {
		for range ticker.C {
			err := c.DumpModel(path)
			if err != nil {
				log.Println("Failed to dump the model ", err)
			} else {
				log.Println("Model was dumped to ", path)
			}
		}
	}()
}
