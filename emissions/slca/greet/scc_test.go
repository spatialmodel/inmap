package greet

import (
	"os"
	"testing"
)

func TestAddSCCs(t *testing.T) {
	f, err := os.Open("default.greet")
	if err != nil {
		t.Fatal(err)
	}
	db := Load(f)

	stationary, err := os.Open("scc/GREET to SCC.csv")
	if err != nil {
		t.Fatal(err)
	}
	vehicles, err := os.Open("scc/GREET vehicle SCC.csv")
	if err != nil {
		panic(err)
	}
	tech, err := os.Open("scc/GREET technology SCC.csv")
	if err != nil {
		panic(err)
	}
	err = db.AddSCCs(stationary, vehicles, tech)
	if err != nil {
		t.Fatal(err)
	}
}
