package jobs

import (
	"fmt"
	"testing"
)

func TestInputHash(t *testing.T) {
	const want = "9692d28a99a12be147229d08b12b1d434f169adca4799a2fa4c279d276b15000"
	if got := fmt.Sprintf("%x", InputHash(42, 1000)); got != want {
		t.Fatalf("got %s; want %s", got, want)
	}
	if InputHash(-1, 23) == InputHash(-12, 3) {
		t.Fatal("ambiguous field boundary")
	}
}
