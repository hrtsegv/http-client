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

// ParseBody parses the input body string into a byte slice.
func ParseBody(bodyStr string) ([]byte, error) {
	bodyStr = strings.TrimSpace(bodyStr)

	if len(bodyStr) == 0 {
		return []byte{}, nil
	}

	// If it's valid JSON, return it as is
	if json.Valid([]byte(bodyStr)) {
		return []byte(bodyStr), nil
	}

	// Fallback to simplified format: {key:value,key2:value2}
	if strings.HasPrefix(bodyStr, "{") && strings.HasSuffix(bodyStr, "}") {
		content := bodyStr[1 : len(bodyStr)-1]
		keyValuePairs := strings.Split(content, ",")
		data := make(map[string]interface{})

		for _, kvPair := range keyValuePairs {
			parts := strings.SplitN(kvPair, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid body format")
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Try to remove surrounding quotes if present (poor man's unquote)
			key = strings.Trim(key, "\"'")
			value = strings.Trim(value, "\"'")

			data[key] = value
		}
		return json.Marshal(data)
	}

	// If neither, just return as raw bytes
	return []byte(bodyStr), nil
}
