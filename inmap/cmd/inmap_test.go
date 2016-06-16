package cmd

import (
	"os"
	"testing"
)

func TestCreateGrid(t *testing.T) {
	if err := Startup("../configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := Grid(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPStaticCreateGrid(t *testing.T) {
	dynamic := false
	createGrid := true
	os.Setenv("InMAPRunType", "static")
	if err := Startup("../configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := Run(dynamic, createGrid); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPStaticLoadGrid(t *testing.T) {
	dynamic := false
	createGrid := false
	os.Setenv("InMAPRunType", "staticLoadGrid")
	if err := Startup("../configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := Run(dynamic, createGrid); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPDynamic(t *testing.T) {
	dynamic := true
	createGrid := false // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic")
	if err := Startup("../configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := Run(dynamic, createGrid); err != nil {
		t.Fatal(err)
	}
}
