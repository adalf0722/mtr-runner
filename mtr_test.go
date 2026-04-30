package main

import (
	"testing"
)

func TestResolveHost_Invalid(t *testing.T) {
	_, err := resolveHost("this.host.does.not.exist.invalid")
	if err == nil {
		t.Fatal("expected error for invalid host")
	}
}

func TestRoundMs(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{1.23456, 1.2},
		{0, 0},
		{99.99, 100.0},
	}
	for _, c := range cases {
		got := roundMs(c.in)
		if got != c.want {
			t.Errorf("roundMs(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSqrtF(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0, 0},
		{4, 2},
		{9, 3},
		{-1, 0},
	}
	for _, c := range cases {
		got := sqrtF(c.in)
		diff := got - c.want
		if diff < -0.001 || diff > 0.001 {
			t.Errorf("sqrtF(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
