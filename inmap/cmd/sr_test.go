package cmd

import (
	"os"
	"testing"
)

func TestSR(t *testing.T) {
	if err := Startup("../configExample.toml"); err != nil {
		t.Fatal(err)
	}
	Config.SROutputFile = "tempSR.ncf"
	begin := 8
	end := 9
	layers := []int{0}
	if err := RunSR(begin, end, layers); err != nil {
		t.Fatal(err)
	}
	os.Remove(Config.SROutputFile)
}

func TestWorkerInit(t *testing.T) {
	if err := Startup("../configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if _, err := InitWorker(); err != nil {
		t.Fatal(err)
	}
}
