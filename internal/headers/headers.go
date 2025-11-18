package headers

import (
	"bytes"
	"fmt"
	"strings"
)

const crlf = "\r\n"

// isValidHeaderKey checks if the header key contains only valid characters according to HTTP spec
func isValidHeaderKey(key string) bool {
	for _, c := range key {
		if (c >= 'A' && c <= 'Z') ||
			(c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') ||
			(c == '!' || c == '#' || c == '$' || c == '%' || c == '&' || c == '\'' ||
				c == '*' || c == '+' || c == '-' || c == '.' || c == '^' || c == '_' ||
				c == '`' || c == '|' || c == '~') {
			continue
		}
		return false
	}
	return true
}

type Headers map[string]string

func NewHeaders() Headers {
	return map[string]string{}
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	idx := bytes.Index(data, []byte(crlf))

	if idx == -1 {
		return 0, false, nil
	}

	if idx == 0 {
		//empty line
		//headers are done
		return 2, true, nil
	}

	parts := bytes.SplitN(data[:idx], []byte(":"), 2)
	if len(parts) != 2 {
		return 0, false, fmt.Errorf("malformed header: missing colon")
	}

	key := string(parts[0])

	if key != strings.TrimRight(key, " ") {
		return 0, false, fmt.Errorf("invalid header name: %s", key)
	}

	value := bytes.TrimSpace(parts[1])
	key = strings.TrimSpace(key)
	if !isValidHeaderKey(key) {
		return 0, false, fmt.Errorf("invalid header name: contains invalid characters: %s", key)
	}

	h.Set(strings.ToLower(key), string(value))
	return idx + 2, false, nil
}

func (h Headers) Set(key, value string) {
	if existingValue, exists := h[key]; exists {
		h[key] = existingValue + ", " + value
	} else {
		h[key] = value
	}
}

func (h Headers) Get(key string) string {
	return h[strings.ToLower(key)]
}

// SetOverride sets a header value, replacing any existing value instead of appending
func (h Headers) SetOverride(key, value string) {
	h[strings.ToLower(key)] = value
}
