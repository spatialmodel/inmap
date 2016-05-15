package main

import (
	"flag"
	"testing"
)

func TestInMAP(t *testing.T) {
	err := flag.Set("config", "configExample.toml")
	if err != nil {
		t.Fatal(err)
	}
	main()
}
