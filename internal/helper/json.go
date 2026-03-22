package helper

import "encoding/json"

const jsonIndent = "  "

// MarshalJSON serializes v to JSON. When pretty is true, output is indented.
func MarshalJSON(v any, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(v, "", jsonIndent)
	}

	return json.Marshal(v)
}

// MarshalJSONIndent serializes v to indented JSON.
func MarshalJSONIndent(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", jsonIndent)
}
