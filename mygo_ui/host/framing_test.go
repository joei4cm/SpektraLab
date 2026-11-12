package host

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	header := map[string]any{"id": 1, "method": "ping", "params": map[string]any{}}
	payload := []byte{1, 2, 3, 4}
	if err := writeFrame(&buf, header, payload); err != nil {
		t.Fatal(err)
	}
	h, p, err := readFrame(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(p, payload) {
		t.Fatalf("payload mismatch: %v", p)
	}
	var got map[string]any
	if err := json.Unmarshal(h, &got); err != nil {
		t.Fatal(err)
	}
	if got["method"] != "ping" {
		t.Fatalf("method=%v", got["method"])
	}
}
