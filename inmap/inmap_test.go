package main

import (
	"testing"

	"github.com/spatialmodel/inmap/inmap/cmd"
)

func TestInMAPStatic(t *testing.T) {
	dynamic := false
	if err := cmd.Startup("configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(dynamic); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPDyanamic(t *testing.T) {
	dynamic := true
	if err := cmd.Startup("configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(dynamic); err != nil {
		t.Fatal(err)
	}
}
