package main

import "testing"

func TestNothing(t *testing.T) {
	if 6 == 5 {
		t.Error("Expected nothing")
	}
}
