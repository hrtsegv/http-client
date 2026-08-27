package utilities

import (
	"encoding/json"
	"testing"
)

func TestParseBody(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "Standard JSON",
			input:   `{"key1":"value1"}`,
			want:    `{"key1":"value1"}`,
			wantErr: false,
		},
		{
			name:    "Plain text body is passed through as-is",
			input:   "not json, just text with a comma, right there",
			want:    "not json, just text with a comma, right there",
			wantErr: false,
		},
		{
			name:    "Empty body",
			input:   "",
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBody(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// We compare JSON strings, but order might differ.
				// For simplicity in this test, we just check if it matches the 'want' exactly
				// or we could unmarshal both and compare maps.
				if string(got) != tt.want {
					t.Errorf("ParseBody() got = %s, want %s", got, tt.want)
				}
			}
		})
	}
}

func TestParseDataFields(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "Multiple fields",
			input: []string{"key1=value1", "key2=value2"},
			want:  map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:  "Value containing an equals sign",
			input: []string{"query=a=b"},
			want:  map[string]string{"query": "a=b"},
		},
		{
			name:    "Missing equals sign",
			input:   []string{"invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDataFields(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDataFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			var gotMap map[string]string
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("ParseDataFields() produced invalid JSON: %v", err)
			}
			for k, v := range tt.want {
				if gotMap[k] != v {
					t.Errorf("ParseDataFields() got[%s] = %v, want %v", k, gotMap[k], v)
				}
			}
		})
	}
}

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "Valid JSON headers",
			input:   []string{`{"Content-Type": "application/json"}`},
			want:    map[string]string{"Content-Type": "application/json"},
			wantErr: false,
		},
		{
			name:    "Standard format headers",
			input:   []string{"Content-Type: application/json", "X-Custom: value"},
			want:    map[string]string{"Content-Type": "application/json", "X-Custom": "value"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHeaders(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHeaders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("ParseHeaders() got[%s] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
