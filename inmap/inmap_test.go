package main

import (
	"testing"

	"github.com/spatialmodel/inmap/inmap/cmd"
)

func TestCreateGrid(t *testing.T) {
	if err := cmd.Startup("configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Grid(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPStaticCreateGrid(t *testing.T) {
	dynamic := false
	createGrid := true
	if err := cmd.Startup("configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(dynamic, createGrid); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPStaticLoadGrid(t *testing.T) {
	dynamic := false
	createGrid := false
	if err := cmd.Startup("configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(dynamic, createGrid); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPDynamic(t *testing.T) {
	dynamic := true
	createGrid := false // this isn't used for the dynamic grid
	if err := cmd.Startup("configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(dynamic, createGrid); err != nil {
		t.Fatal(err)
	}
}
