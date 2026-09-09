package utilities

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
)

// ParseHeaders parses the input headers slice into a map.
func ParseHeaders(headers []string) (map[string]string, error) {
	headerMap := make(map[string]string)

	for _, h := range headers {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}

		// Support both JSON format and "Key: Value" format
		if strings.HasPrefix(h, "{") && strings.HasSuffix(h, "}") {
			var m map[string]string
			if err := json.Unmarshal([]byte(h), &m); err == nil {
				for k, v := range m {
					headerMap[k] = v
				}
				continue
			}
		}

		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header format: %s. Expected 'Key: Value'", h)
		}
		headerMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	return headerMap, nil
}

// ParseBody resolves a body argument into raw bytes, forwarded to the
// request as-is. bodyStr may be raw text/JSON, "@path" to read the body
// from a file, or "-" to read it from stdin.
func ParseBody(bodyStr string) ([]byte, error) {
	bodyStr = strings.TrimSpace(bodyStr)

	switch {
	case bodyStr == "":
		return []byte{}, nil
	case bodyStr == "-":
		return io.ReadAll(os.Stdin)
	case strings.HasPrefix(bodyStr, "@"):
		return os.ReadFile(strings.TrimPrefix(bodyStr, "@"))
	default:
		return []byte(bodyStr), nil
	}
}

// KeyValue is one parsed "key=value" argument.
type KeyValue struct {
	Key   string
	Value string
}

// ParseKeyValuePairs splits repeated "key=value" arguments, keeping them in
// the order given and leaving any further '=' in the value. flagName names
// the flag being parsed so errors can point at it.
func ParseKeyValuePairs(fields []string, flagName string) ([]KeyValue, error) {
	pairs := make([]KeyValue, 0, len(fields))

	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}

		key, value, ok := strings.Cut(f, "=")
		if !ok {
			return nil, fmt.Errorf("invalid %s value: %s. Expected 'key=value'", flagName, f)
		}

		pairs = append(pairs, KeyValue{Key: strings.TrimSpace(key), Value: value})
	}

	return pairs, nil
}

// ParseFormFields builds an application/x-www-form-urlencoded body from
// repeated "key=value" fields. A repeated key adds another value, as HTML
// forms do for checkboxes and multi-selects.
func ParseFormFields(fields []string) ([]byte, error) {
	pairs, err := ParseKeyValuePairs(fields, "--form")
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	for _, pair := range pairs {
		values.Add(pair.Key, pair.Value)
	}

	return []byte(values.Encode()), nil
}

// ParseDataFields builds a JSON object body from repeated "key=value"
// fields, e.g. from a -d/--data flag.
func ParseDataFields(fields []string) ([]byte, error) {
	pairs, err := ParseKeyValuePairs(fields, "--data")
	if err != nil {
		return nil, err
	}

	data := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		data[pair.Key] = pair.Value
	}

	return json.Marshal(data)
}
