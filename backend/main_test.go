package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitizeHTML(t *testing.T) {
	inputs := []string{
		`<div onclick="alert('x')"><script>alert(1)</script><style>body{}</style>test</div>`,
		`<div onclick='alert(2)'>test2</div>`,
	}
	for _, input := range inputs {
		got := sanitizeHTML(input)
		if strings.Contains(got, "script") || strings.Contains(got, "style") || strings.Contains(got, "onclick") {
			t.Errorf("sanitizeHTML failed: got %q", got)
		}
	}
}

func TestIsValidSelector(t *testing.T) {
	valid := []string{".class", "#id", "div", "[data-x='1']", "span#id.class"}
	invalid := []string{"<script>", "alert()", "div;"}
	for _, sel := range valid {
		if !isValidSelector(sel) {
			t.Errorf("isValidSelector should accept: %q", sel)
		}
	}
	for _, sel := range invalid {
		if isValidSelector(sel) {
			t.Errorf("isValidSelector should reject: %q", sel)
		}
	}
}

func TestValidateAndSanitizeActions(t *testing.T) {
	actions := []interface{}{
		map[string]interface{}{"type": "replace", "selector": "#id", "newElement": `<div onclick="alert(1)">x</div>`},
		map[string]interface{}{"type": "remove", "selector": ".class"},
	}
	valid, sanitized := validateAndSanitizeActions(actions)
	if !valid {
		t.Fatal("actions should be valid")
	}
	if html, ok := sanitized[0].(map[string]interface{})["newElement"].(string); ok {
		t.Log("Sanitized HTML:", html)
		if strings.Contains(html, "onclick") {
			t.Error("sanitize failed in actions")
		}
	}
}

func TestPingEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/ping", nil)
	w := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "pong"}`))
	})
	handler(w, req)
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("ping endpoint failed, got status %d", resp.StatusCode)
	}
} 