package model

import (
	"encoding/json"
	"testing"
)

func TestDateJSONRoundTrip(t *testing.T) {
	type wrapper struct {
		D Date `json:"d"`
	}
	w := wrapper{D: MustParseDate("2026-06-17")}
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"d":"2026-06-17"}` {
		t.Fatalf("marshal = %s", b)
	}
	var back wrapper
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.D != w.D {
		t.Fatalf("round trip mismatch: %v", back.D)
	}
}

func TestDateJSONZeroAndNull(t *testing.T) {
	b, err := json.Marshal(Date{})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "null" {
		t.Fatalf("zero date should marshal to null, got %s", b)
	}
	var d Date
	for _, in := range []string{`null`, `""`, `"2026-01-02"`} {
		if err := json.Unmarshal([]byte(in), &d); err != nil {
			t.Fatalf("unmarshal %s: %v", in, err)
		}
	}
	if err := json.Unmarshal([]byte(`"nonsense"`), &d); err == nil {
		t.Fatal("expected error unmarshaling bad date")
	}
}

func TestMustParseDatePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("MustParseDate should panic on invalid input")
		}
	}()
	_ = MustParseDate("not-a-date")
}
