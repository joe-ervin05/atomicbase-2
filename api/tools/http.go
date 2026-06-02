package tools

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/atombasedev/atombase/config"
)

// ParseHeaderCommas splits comma-separated header values into individual strings.
func ParseHeaderCommas(strs []string) []string {
	out := make([]string, 0, len(strs))

	for _, s := range strs {
		for _, part := range strings.Split(s, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}

	return out
}

// LimitBody applies the configured request body size limit to the request body.
func LimitBody(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.Body == nil {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, config.Cfg.MaxRequestBody)
}

// DecodeJSON decodes a JSON request body into the provided target.
func DecodeJSON(body io.Reader, target any) error {
	return json.NewDecoder(body).Decode(target)
}
