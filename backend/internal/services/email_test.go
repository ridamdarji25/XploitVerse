package services

import (
	"testing"
)

func TestEmailServiceIsConfigured(t *testing.T) {
	unconfigured := NewEmailService(SMTPConfig{})
	if unconfigured.IsConfigured() {
		t.Errorf("Expected IsConfigured to be false for empty config")
	}

	configured := NewEmailService(SMTPConfig{
		Host:     "smtp.gmail.com",
		Port:     "587",
		Username: "test@example.com",
		Password: "password",
	})
	if !configured.IsConfigured() {
		t.Errorf("Expected IsConfigured to be true for complete config")
	}
}

func TestRenderResetEmail(t *testing.T) {
	html, err := renderResetEmail("https://example.com/reset-password/abc123token", "HackerZero")
	if err != nil {
		t.Fatalf("renderResetEmail failed: %v", err)
	}

	if len(html) == 0 {
		t.Errorf("Expected rendered HTML not to be empty")
	}

	if !containsSubstring(html, "HackerZero") {
		t.Errorf("Expected email to contain username 'HackerZero'")
	}

	if !containsSubstring(html, "https://example.com/reset-password/abc123token") {
		t.Errorf("Expected email to contain reset URL")
	}
}

func TestRenderOTPEmail(t *testing.T) {
	html, err := renderOTPEmail("849201", "CipherChief")
	if err != nil {
		t.Fatalf("renderOTPEmail failed: %v", err)
	}

	if len(html) == 0 {
		t.Errorf("Expected rendered HTML not to be empty")
	}

	if !containsSubstring(html, "849201") {
		t.Errorf("Expected email to contain OTP 849201")
	}

	if !containsSubstring(html, "CipherChief") {
		t.Errorf("Expected email to contain username 'CipherChief'")
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || searchSub(s, sub))
}

func searchSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
