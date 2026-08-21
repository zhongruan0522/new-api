package jsonx

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	in := payload{Name: "relay", Count: 3}
	data, err := Marshal(in)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	var out payload
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if out != in {
		t.Fatalf("round trip mismatch: got %+v, want %+v", out, in)
	}
}

func TestUnmarshalPropagatesInvalidJsonError(t *testing.T) {
	var v map[string]any
	if err := Unmarshal([]byte(`{invalid`), &v); err == nil {
		t.Fatal("Unmarshal must propagate invalid JSON errors instead of swallowing them")
	}
}

func TestUnmarshalJsonStrDecodesSameAsUnmarshal(t *testing.T) {
	src := `{"ok":true,"n":2}`
	var fromStr, fromBytes struct {
		OK bool `json:"ok"`
		N  int  `json:"n"`
	}
	if err := UnmarshalJsonStr(src, &fromStr); err != nil {
		t.Fatalf("UnmarshalJsonStr returned error: %v", err)
	}
	if err := Unmarshal([]byte(src), &fromBytes); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if fromStr != fromBytes {
		t.Fatalf("UnmarshalJsonStr decoded %+v, Unmarshal decoded %+v", fromStr, fromBytes)
	}
}

func TestStringToByteSliceExposesReadOnlyView(t *testing.T) {
	s := "hello"
	b := StringToByteSlice(s)
	if string(b) != s {
		t.Fatalf("StringToByteSlice view = %q, want %q", b, s)
	}
	var decoded map[string]int
	if err := json.Unmarshal(StringToByteSlice(`{"a":1}`), &decoded); err != nil {
		t.Fatalf("unmarshal through StringToByteSlice view returned error: %v", err)
	}
	if decoded["a"] != 1 {
		t.Fatalf("decoded %v, want map[a:1]", decoded)
	}
}

func TestDecodeJson(t *testing.T) {
	var got []int
	if err := DecodeJson(strings.NewReader(`[1,2,3]`), &got); err != nil {
		t.Fatalf("DecodeJson returned error: %v", err)
	}
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("DecodeJson decoded %v, want [1 2 3]", got)
	}
}

func TestValid(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{`{"a":1}`, true},
		{`[1,2,3]`, true},
		{`"text"`, true},
		{`42`, true},
		{`null`, true},
		{``, false},
		{`{invalid`, false},
		{`{"a":}`, false},
		{`[1,2,`, false},
		{`nul`, false},
	}
	for _, c := range cases {
		if got := Valid([]byte(c.raw)); got != c.want {
			t.Errorf("Valid(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
}

func TestGetJsonType(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{`  {"a":1}  `, "object"},
		{`[1,2]`, "array"},
		{`"text"`, "string"},
		{`true`, "boolean"},
		{`false`, "boolean"},
		{`null`, "null"},
		{`42`, "number"},
		{`-1.5`, "number"},
		{`   `, "unknown"},
		{``, "unknown"},
	}
	for _, c := range cases {
		if got := GetJsonType(json.RawMessage(c.raw)); got != c.want {
			t.Errorf("GetJsonType(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}
