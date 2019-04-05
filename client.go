package govw

import (
	"bufio"
	"fmt"
	"github.com/fatih/pool"
	"net"
	"strings"
)

// endOfLine is represent byte code for symbol of end of line: `\n`
const endOfLine = 10

// Predict contain result of prediction
type Prediction struct {
	Value float64
	Tag   string
}

type VWClient struct {
	Pool pool.Pool
}

func NewClient() *VWClient {
	return &VWClient{}
}

func (th *VWClient) Connect(host string, port int, maxConnections int) error {
	// create a factory() to be used with channel based pool
	factory := func() (net.Conn, error) {
		return CreateTCPConn(host, port)
	}

	initialCap := maxConnections / 2
	var err error
	if th.Pool, err = pool.NewChannelPool(initialCap, maxConnections, factory); err != nil {
		return fmt.Errorf("failed to create tcp pull: %s", err)
	}

	return nil
}

func (th *VWClient) Disconnect() error {
	th.Pool.Close()

	return nil
}

// send data to VW daemon and read an answer
// return list of raw vw responses.
func (th *VWClient) ask(waitResponse bool, requests ...string) ([]string, error) {
	size := len(requests)
	responses := make([]string, size)

	data := []byte(strings.Join(requests, "\n"))
	data = append(data, endOfLine)

	conn, err := th.Pool.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection from the pool: %s", err)
	}
	// this doesn't close the underlying connection instead it's putting it back to the pool
	defer conn.Close()

	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write to connection: %s", err)
	}

	if !waitResponse {
		return nil, nil
	}

	reader := bufio.NewReader(conn)
	for i := 0; i < size; i++ {
		res, err := reader.ReadString('\n')
		if err != nil && err.Error() != "EOF" {
			return responses, fmt.Errorf("failed to read response: %s", err)
		}

		responses[i] = res
	}

	return responses, nil
}

// Predict method get predictions strings then send data to VW daemon
// for getting prediction result and return list of predictions result.
func (th *VWClient) Predict(pData ...string) ([]*Prediction, error) {
	lines, err := th.ask(true, pData...)
	if err != nil {
		return nil, fmt.Errorf("failed to ask vw: %s", err)
	}

	result := make([]*Prediction, len(pData))

	for i, line := range lines {
		r, err := ParsePredictResult(&line)
		if err != nil {
			return result, err
		}

		result[i] = r
	}

	return result, nil
}

func (th *VWClient) DumpModel(path string) error {
	_, err := th.ask(false, "save_"+path)
	if err != nil {
		return fmt.Errorf("failed to ask vw: %s", err)
	}

	return nil
}
