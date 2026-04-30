package main

import (
	"testing"
)

func TestParseTraceroute_basic(t *testing.T) {
	input := `traceroute to 8.8.8.8 (8.8.8.8), 5 hops max, 40 byte packets
 1  192.168.0.1 (192.168.0.1)  3.088 ms  1.555 ms  1.396 ms
 2  10.0.0.1 (10.0.0.1)  12.750 ms  11.783 ms  11.661 ms
 3  * * *
`
	hops := parseTraceroute(input)
	if len(hops) != 3 {
		t.Fatalf("expected 3 hops, got %d", len(hops))
	}
	if hops[0].Host != "192.168.0.1" {
		t.Errorf("hop 1 host = %q, want 192.168.0.1", hops[0].Host)
	}
	if hops[0].Loss != 0 {
		t.Errorf("hop 1 loss = %v, want 0", hops[0].Loss)
	}
	if hops[2].Host != "???" {
		t.Errorf("hop 3 host = %q, want ???", hops[2].Host)
	}
	if hops[2].Loss != 100 {
		t.Errorf("hop 3 loss = %v, want 100", hops[2].Loss)
	}
}

func TestParseTraceroute_ecmp(t *testing.T) {
	// hop 4 has ECMP continuation lines — all RTTs should be collected into one hop
	input := `traceroute to 8.8.8.8 (8.8.8.8), 5 hops max, 40 byte packets
 1  192.168.0.1 (192.168.0.1)  3.0 ms  2.0 ms  1.0 ms
 4  r56-186.seed.net.tw (139.175.56.186)  14.0 ms
    r57-150.seed.net.tw (139.175.57.150)  17.0 ms
    r56-50.seed.net.tw (139.175.56.50)  12.0 ms  10.0 ms
`
	hops := parseTraceroute(input)
	if len(hops) != 2 {
		t.Fatalf("expected 2 hops (hop 1 and hop 4), got %d", len(hops))
	}
	h4 := hops[1]
	if h4.Count != 4 {
		t.Errorf("expected hop 4, got %d", h4.Count)
	}
	// 4 RTT samples total across 3 continuation lines
	if h4.Snt < 4 {
		t.Errorf("expected at least 4 samples for ECMP hop, got Snt=%d", h4.Snt)
	}
	if h4.Loss != 0 {
		t.Errorf("expected 0 loss for ECMP hop, got %.1f%%", h4.Loss)
	}
}

func TestParseTraceroute_partialLoss(t *testing.T) {
	// 3 probes, 1 timeout = 33.3% loss
	input := `traceroute to 8.8.8.8 (8.8.8.8), 5 hops max, 40 byte packets
 1  192.168.0.1 (192.168.0.1)  3.0 ms  * 2.0 ms
`
	hops := parseTraceroute(input)
	if len(hops) != 1 {
		t.Fatalf("expected 1 hop, got %d", len(hops))
	}
	if hops[0].Loss == 0 || hops[0].Loss == 100 {
		t.Errorf("expected partial loss, got %v%%", hops[0].Loss)
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
