package helper

import (
	"encoding/json"
	"testing"
)

func TestMarshalJSON(t *testing.T) {
	v := map[string]int{"a": 1}

	tests := []struct {
		name   string
		pretty bool
		want   string
	}{
		{"compact", false, `{"a":1}`},
		{"pretty", true, "{\n  \"a\": 1\n}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalJSON(v, tt.pretty)
			if err != nil {
				t.Fatalf("MarshalJSON() error: %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("MarshalJSON() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestMarshalJSONIndent(t *testing.T) {
	v := map[string]int{"a": 1}
	want := "{\n  \"a\": 1\n}"

	got, err := MarshalJSONIndent(v)
	if err != nil {
		t.Fatalf("MarshalJSONIndent() error: %v", err)
	}

	if string(got) != want {
		t.Errorf("MarshalJSONIndent() = %q, want %q", string(got), want)
	}
}

func TestMarshalJSONError(t *testing.T) {
	_, err := MarshalJSON(make(chan int), false)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestMarshalJSONIndentMatchesStdlib(t *testing.T) {
	v := []string{"hello", "world"}

	got, err := MarshalJSONIndent(v)
	if err != nil {
		t.Fatalf("MarshalJSONIndent() error: %v", err)
	}

	want, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("MarshalJSONIndent() = %q, want %q", string(got), string(want))
	}
}
