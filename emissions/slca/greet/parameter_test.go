package greet

import "testing"

func TestParameterColumn(t *testing.T) {
	type test struct {
		in, out string
	}
	tests := []test{
		test{in: "1", out: "A"},
		test{in: "34", out: "AH"},
		test{in: "16384", out: "XFD"},
	}
	for _, tt := range tests {
		got := numberToColumn(tt.in)
		if got != tt.out {
			t.Errorf("FAIL: got %s, want %s", got, tt.out)
		}
	}
}
