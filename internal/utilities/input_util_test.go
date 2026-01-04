package utilities

import (
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
			name:    "Simple JSON-like",
			input:   "{key1:value1,key2:value2}",
			want:    `{"key1":"value1","key2":"value2"}`,
			wantErr: false,
		},
		{
			name:    "Standard JSON (currently broken)",
			input:   `{"key1":"value1"}`,
			want:    `{"key1":"value1"}`,
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
