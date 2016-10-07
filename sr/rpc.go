package sr

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Cluster manages and distributes work to a group of workers.
type Cluster struct {
	requestChan     chan *Request
	command, logDir string
	exitService     string
	initData        interface{}
	rpcPort         string

	// StartupTime specifies how long a worker is expected to take to initialize.
	// The default is 3 minutes.
	StartupTime time.Duration
}

// NewCluster creates a new cluster of workers where log output will be
// directed to logDir. exitService is the name of the
// RPC service that should be called to shut down each worker, which will not be
// be sent any input data. RPCPort is the port over which RPC communication
// will occur.
func NewCluster(command, logDir, exitService string, RPCPort string) *Cluster {
	return &Cluster{
		requestChan: make(chan *Request),
		command:     command,
		logDir:      logDir,
		rpcPort:     RPCPort,
		StartupTime: time.Minute * 3,
	}
}

// Shutdown sends a signal to the workers to run the exitService after all
// existing requests have finished processing.
func (c *Cluster) Shutdown() {
	close(c.requestChan)
}

// NewWorker creates a new worker at addr using the external ssh command
// and prepares it to receive direction through RPC calls.
func (c *Cluster) NewWorker(addr string) error {
	err := c.spawnSlave(addr, c.StartupTime)
	if err != nil {
		return err
	}

	client, err := rpc.DialHTTP("tcp", addr+":"+c.rpcPort)
	if err != nil {
		return fmt.Errorf("while dialing %v: %v", addr, err)
	}
	go func() {
		for req := range c.requestChan {
			req.err = client.Call(req.service, req.requestPayload, &req.resultPayload)
			req.returnChan <- req
		}
		// kill the slave after we're done with it.
		client.Call(c.exitService, 0, 0)
	}()
	return nil
}

// spawnSlave executes an external process to spawn a slave
// at the address "addr" using the external "ssh" command.
// Stdout from the slave is routed
// to the directory "logDir".
func (c *Cluster) spawnSlave(addr string, timeout time.Duration) error {
	log.Println("Spawning slave ", addr)
	cmd := exec.Command("ssh", addr, c.command)

	f, err := os.Create(filepath.Join(c.logDir, addr+".log"))
	if err != nil {
		return err
	}
	cmd.Stdout = f
	cmd.Stderr = f

	// After a Session is created, we execute a single command on
	// the remote side using the Run method.
	go func() {
		if err := cmd.Run(); err != nil {
			if err.Error() == "signal: killed" {
				log.Printf("worker %v expected error: %v", addr, err.Error())
			} else {
				panic(fmt.Errorf("Slave %v error: %v", addr, err.Error()))
			}
		}
	}()
	time.Sleep(timeout) // wait for a while for it to get started
	return nil
}

// PBSNodes reads the contents of $PBS_NODEFILE and returns a list of
// unique nodes.
func PBSNodes() ([]string, error) {
	fname := os.ExpandEnv("$PBS_NODEFILE")
	if fname == "" {
		return nil, fmt.Errorf("$PBS_NODEFILE not defined")
	}
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(f)
	lines, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	slavesMap := make(map[string]string)
	for _, l := range lines {
		slavesMap[l[0]] = ""
	}
	var slaves []string
	for s := range slavesMap {
		slaves = append(slaves, s)
	}
	return slaves, nil
}
