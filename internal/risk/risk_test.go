package risk

import "testing"

func TestRepeatable(t *testing.T) {
	first, err := Run(42, 1000)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run(42, 1000)
	if err != nil || second != first || first.Losses <= 0 || first.Losses >= first.Trials {
		t.Fatalf("unexpected runs: %+v %+v, %v", first, second, err)
	}
}

func TestBounds(t *testing.T) {
	for _, n := range []int{-1, 0, MaxTrials + 1} {
		if _, err := Run(1, n); err == nil {
			t.Errorf("accepted %d trials", n)
		}
	}
	for _, n := range []int{1, MaxTrials} {
		if _, err := Run(1, n); err != nil {
			t.Fatal(err)
		}
	}
}
