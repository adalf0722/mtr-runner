package main

import (
	"testing"
)

func TestEncodeRoundtrip(t *testing.T) {
	original := `{"report":{"mtr":{"dst":"google.com"}}}`
	encoded, err := encodeData(original)
	if err != nil {
		t.Fatalf("encodeData error: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("encoded string is empty")
	}
	for _, c := range encoded {
		if c == '+' || c == '/' || c == '=' {
			t.Fatalf("encoded contains URL-unsafe char: %c", c)
		}
	}
}

func TestEncodeShorterThanInput(t *testing.T) {
	large := `{"data":"` + string(make([]byte, 1000)) + `"}`
	encoded, err := encodeData(large)
	if err != nil {
		t.Fatalf("encodeData error: %v", err)
	}
	if len(encoded) >= len(large) {
		t.Errorf("expected encoded (%d) < original (%d)", len(encoded), len(large))
	}
}
