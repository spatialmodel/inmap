package main

import (
	"flag"
	"testing"
)

func TestInMAP(t *testing.T) {
	err := flag.Set("config", "configExample.json")
	if err != nil {
		t.Fatal(err)
	}
	main()
}
