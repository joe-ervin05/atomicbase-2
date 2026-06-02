package tools

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/atombasedev/atombase/config"
)

func TestParseHeaderCommas(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "single value",
			input: []string{"operation=select"},
			want:  []string{"operation=select"},
		},
		{
			name:  "comma separated values",
			input: []string{"operation=select, count=exact"},
			want:  []string{"operation=select", "count=exact"},
		},
		{
			name:  "multiple headers and empty parts",
			input: []string{"operation=insert, , on-conflict=replace", "count=exact"},
			want:  []string{"operation=insert", "on-conflict=replace", "count=exact"},
		},
		{
			name:  "trims whitespace",
			input: []string{"  one  , two ", " three "},
			want:  []string{"one", "two", "three"},
		},
		{
			name:  "empty input",
			input: nil,
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseHeaderCommas(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d parts, got %d (%v)", len(tt.want), len(got), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("expected part %d to be %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}

func TestLimitBody_EnforcesMaxRequestBody(t *testing.T) {
	originalLimit := config.Cfg.MaxRequestBody
	config.Cfg.MaxRequestBody = 8
	defer func() {
		config.Cfg.MaxRequestBody = originalLimit
	}()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		LimitBody(w, r)
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/platform/definitions", strings.NewReader("123456789"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "request body too large") {
		t.Fatalf("expected body too large error, got %q", rec.Body.String())
	}
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	var got payload
	if err := DecodeJSON(strings.NewReader(`{"name":"alice"}`), &got); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Name != "alice" {
		t.Fatalf("expected name alice, got %q", got.Name)
	}

	if err := DecodeJSON(strings.NewReader(`{"name":`), &got); err == nil {
		t.Fatal("expected decode error")
	}
}
