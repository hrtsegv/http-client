package utilities

import (
	"encoding/json"
	"fmt"
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

// ParseBody trims the input body string and returns it as raw bytes. The
// body is forwarded to the request as-is, whether it's JSON, XML, plain
// text, or anything else.
func ParseBody(bodyStr string) ([]byte, error) {
	bodyStr = strings.TrimSpace(bodyStr)

	if len(bodyStr) == 0 {
		return []byte{}, nil
	}

	return []byte(bodyStr), nil
}

// ParseDataFields builds a JSON object body from repeated "key=value"
// fields, e.g. from a -d/--data flag.
func ParseDataFields(fields []string) ([]byte, error) {
	data := make(map[string]string, len(fields))

	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}

		key, value, ok := strings.Cut(f, "=")
		if !ok {
			return nil, fmt.Errorf("invalid data field: %s. Expected 'key=value'", f)
		}

		data[strings.TrimSpace(key)] = value
	}

	return json.Marshal(data)
}
