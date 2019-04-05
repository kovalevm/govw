package govw

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// VWModel contain information about VW model file
// If `Updatable` field is `true`, the system will be track of the
// changes model file and restart the daemon if necessary
type VWModel struct {
	Path      string
	ModTime   time.Time
	Updatable bool
}

// VWDaemon contain information about VW daemon
type VWDaemon struct {
	BinPath  string
	Port     [2]int
	Children int
	Model    VWModel
	Test     bool
	VwOpts   string
}

// NewDaemon method return instance of new Vowpal Wabbit daemon
func NewDaemon(
	binPath string,
	ports [2]int,
	children int,
	modelPath string,
	test bool,
	vwOpts string,
) (*VWDaemon, error) {
	info, err := os.Stat(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check vw model file: %s", err)
	}

	return &VWDaemon{
		BinPath:  binPath,
		Port:     ports,
		Children: children,
		Model:    VWModel{modelPath, info.ModTime(), false},
		Test:     test,
		VwOpts:   vwOpts,
	}, nil
}

// Run method send command for starting new VW daemon.
func (vw *VWDaemon) Run() error {
	if vw.IsNotDead(3, 200) {
		vw.Stop()
	}

	cmd := fmt.Sprintf("vw --daemon --threads --quiet --port %d --num_children %d", vw.Port[0], vw.Children)

	if vw.Model.Path != "" {
		cmd += fmt.Sprintf(" -i  %s", vw.Model.Path)
	}

	if vw.Test {
		cmd += " -t"
	}

	if vw.VwOpts != "" {
		cmd += " " + vw.VwOpts
	}

	if _, err := runCommand(cmd, true); err != nil {
		return fmt.Errorf("failed to execute vw command: %s", err)
	}

	if !vw.IsExist(5, 500) {
		return fmt.Errorf("failed to start vw daemon")
	}

	log.Printf("Vowpal wabbit daemon is running on port: %d", vw.Port[0])

	return nil
}

// Stop current daemon
func (vw *VWDaemon) Stop() error {
	log.Println("Try stop daemon on port:", vw.Port[0])

	cmd := fmt.Sprintf("pkill -9 -f \"vw.*--port %d\"", vw.Port[0])
	if _, err := runCommand(cmd, true); err != nil {
		return fmt.Errorf("failed to execute 'pkill' command: %s", err)
	}

	for i := 0; i < 5; i++ {
		if vw.IsNotDead(10, 500) {
			log.Println("Failed to stop daemon! Try №", i+1)
			cmd := fmt.Sprintf("pkill -9 -f \"vw.*--port %d\"", vw.Port[0])
			if _, err := runCommand(cmd, true); err != nil {
				return fmt.Errorf("failed to execute 'pkill' command: %s", err)
			}
		} else {
			break
		}
	}
	log.Println("Stopped VW daemon on port:", vw.Port[0])

	return nil
}

func (vw *VWDaemon) WorkersCount() (int, error) {
	cmd := fmt.Sprintf("pgrep -f 'vw.*--port %d' | wc -l", vw.Port[0])
	res, err := runCommand(cmd, false)
	if err != nil {
		return 0, err
	}
	count, err := strconv.Atoi(strings.Trim(string(res), "\n"))
	if err != nil {
		return 0, err
	}

	// We should subtract 1 from count, to get clear result without
	// side effect of using `sh -c` command in `exec.Command`.
	return count - 1, nil
}

// IsNotDead method checks if VW daemon and all of his children is running.
// You should define count of tries and delay in milliseconds between each try.
func (vw *VWDaemon) IsNotDead(tries int, delay int) bool {
	var count int
	var err error

	for i := 0; i < tries; i++ {
		count, err = vw.WorkersCount()

		if count > 0 {
			return true
		}

		time.Sleep(time.Millisecond * time.Duration(delay))
	}
	if err != nil {
		log.Fatal("Can't getting VW workers count.", err)
	}

	return false
}

// IsExist method checks if VW daemon and all of his children is running.
// You should define count of tries and delay in milliseconds between each try.
func (vw *VWDaemon) IsExist(tries int, delay int) bool {
	var count int
	var err error

	for i := 0; i < tries; i++ {
		count, err = vw.WorkersCount()

		// We add 1 to `vw.children`, because we still have the parent process.
		if count == vw.Children+1 {
			return true
		}

		time.Sleep(time.Millisecond * time.Duration(delay))
	}
	if err != nil {
		log.Fatal("Can't getting VW workers count.", err)
	}

	return false
}

// IsChanged method checks whether the model file has been modified.
func (model *VWModel) IsChanged() (bool, error) {
	info, err := os.Stat(model.Path)
	if err != nil {
		log.Println(err)
		return false, err
	}

	if model.ModTime != info.ModTime() {
		return true, nil
	}

	return false, nil
}
