package trailer

import (
	"strings"
	"testing"
)

// --- FormatTrailer tests ---

func TestFormatTrailer(t *testing.T) {
	result := FormatTrailer("Assisted-by", "Kiro/2.32.0")
	expected := "Assisted-by: Kiro/2.32.0"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

// --- BuildTrailerValue tests ---

func TestBuildTrailerValue(t *testing.T) {
	tests := []struct {
		tool         string
		model        string
		includeModel bool
		expected     string
	}{
		{"Kiro", "claude-opus-4.6", true, "Kiro (claude-opus-4.6)"},
		{"Kiro", "claude-opus-4.6", false, "Kiro"},
		{"Claude Code", "", true, "Claude Code"},
		{"Claude Code", "", false, "Claude Code"},
		{"Gemini CLI", "gemini-3-pro", true, "Gemini CLI (gemini-3-pro)"},
	}

	for _, tt := range tests {
		got := BuildTrailerValue(tt.tool, tt.model, tt.includeModel)
		if got != tt.expected {
			t.Errorf("BuildTrailerValue(%q, %q, %v) = %q, want %q",
				tt.tool, tt.model, tt.includeModel, got, tt.expected)
		}
	}
}

// --- HasTrailer tests ---

func TestHasTrailer(t *testing.T) {
	msg := "feat: add login\n\nAssisted-by: Kiro/2.32.0\n"

	if !HasTrailer(msg, "Assisted-by") {
		t.Error("should find existing trailer")
	}
	if HasTrailer(msg, "Signed-off-by") {
		t.Error("should not find non-existent trailer")
	}
}

func TestHasTrailerEmptyMessage(t *testing.T) {
	if HasTrailer("", "Assisted-by") {
		t.Error("empty message should not have any trailer")
	}
}

// --- AddTrailer tests ---

func TestAddTrailerSimpleMessage(t *testing.T) {
	msg := "feat: add login"
	result := AddTrailer(msg, "Assisted-by", "Kiro/2.32.0")

	// Should have blank line separator then trailer
	expected := "feat: add login\n\nAssisted-by: Kiro/2.32.0\n"
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestAddTrailerMessageWithBody(t *testing.T) {
	msg := "feat: add login\n\nImplemented OAuth2 flow."
	result := AddTrailer(msg, "Assisted-by", "Kiro/2.32.0")

	expected := "feat: add login\n\nImplemented OAuth2 flow.\n\nAssisted-by: Kiro/2.32.0\n"
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestAddTrailerExistingTrailers(t *testing.T) {
	msg := "feat: add login\n\nSigned-off-by: Malay Kumar"
	result := AddTrailer(msg, "Assisted-by", "Kiro/2.32.0")

	// Should append on next line — no blank line between trailers
	expected := "feat: add login\n\nSigned-off-by: Malay Kumar\nAssisted-by: Kiro/2.32.0\n"
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestAddTrailerMultipleExistingTrailers(t *testing.T) {
	msg := "feat: add login\n\nSigned-off-by: Malay Kumar\nReviewed-by: Alice"
	result := AddTrailer(msg, "Assisted-by", "Kiro/2.32.0")

	expected := "feat: add login\n\nSigned-off-by: Malay Kumar\nReviewed-by: Alice\nAssisted-by: Kiro/2.32.0\n"
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestAddTrailerIdempotent(t *testing.T) {
	msg := "feat: add login"

	// Add once
	result := AddTrailer(msg, "Assisted-by", "Kiro/2.32.0")

	// Add again — should not duplicate
	result2 := AddTrailer(result, "Assisted-by", "Kiro/2.32.0")

	if result != result2 {
		t.Errorf("should be idempotent.\nfirst:\n%q\nsecond:\n%q", result, result2)
	}

	// Count occurrences
	count := strings.Count(result2, "Assisted-by")
	if count != 1 {
		t.Errorf("trailer should appear once, found %d times", count)
	}
}

func TestAddTrailerEmptyMessage(t *testing.T) {
	result := AddTrailer("", "Assisted-by", "Kiro/2.32.0")
	if result != "" {
		t.Errorf("empty message should stay empty, got %q", result)
	}
}

func TestAddTrailerWhitespaceOnlyMessage(t *testing.T) {
	result := AddTrailer("   \n\n  ", "Assisted-by", "Kiro/2.32.0")
	if strings.Contains(result, "Assisted-by") {
		t.Error("whitespace-only message should not get a trailer")
	}
}

func TestAddTrailerTrailingNewlines(t *testing.T) {
	// Message with trailing newlines — should be cleaned up
	msg := "feat: add login\n\n\n"
	result := AddTrailer(msg, "Assisted-by", "Kiro/2.32.0")

	// Should not have excessive blank lines
	if strings.Contains(result, "\n\n\n") {
		t.Error("should not have triple newlines")
	}
	if !strings.Contains(result, "Assisted-by: Kiro/2.32.0") {
		t.Error("trailer should be present")
	}
}

func TestAddTrailerBodyWithTrailers(t *testing.T) {
	// Full realistic message: subject + body + existing trailers
	msg := "feat(auth): add OAuth login\n\nImplemented OAuth2 flow with Google.\nAdded refresh token handling.\n\nSigned-off-by: Malay Kumar"
	result := AddTrailer(msg, "Assisted-by", "Kiro/2.32.0")

	// Body should be intact, trailer appended to trailer block
	if !strings.Contains(result, "Implemented OAuth2 flow") {
		t.Error("body should be preserved")
	}
	if !strings.Contains(result, "Signed-off-by: Malay Kumar\nAssisted-by: Kiro/2.32.0") {
		t.Error("trailer should be appended to existing trailer block")
	}
}

func TestAddMultipleTrailers(t *testing.T) {
	msg := "feat: add login"

	// Add two different trailers
	msg = AddTrailer(msg, "Assisted-by", "Kiro/2.32.0")
	msg = AddTrailer(msg, "Commit-Mode", "AI Assisted")

	if !strings.Contains(msg, "Assisted-by: Kiro/2.32.0\nCommit-Mode: AI Assisted") {
		t.Errorf("both trailers should be consecutive, got:\n%s", msg)
	}
}

// --- looksLikeTrailer tests ---

func TestLooksLikeTrailer(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"Signed-off-by: Malay Kumar", true},
		{"Assisted-by: Kiro/2.32.0", true},
		{"Co-authored-by: Claude", true},
		{"Refs: #123", true},
		{"Change-Id: I9a8d7", true},
		{"This is a normal sentence.", false},
		{"No colon here", false},
		{"", false},
		{" : empty key", false},
		{"Has Space: value", false}, // space in key
		{"123-numeric: value", false}, // starts with number
	}

	for _, tt := range tests {
		got := looksLikeTrailer(tt.line)
		if got != tt.expected {
			t.Errorf("looksLikeTrailer(%q) = %v, want %v", tt.line, got, tt.expected)
		}
	}
}
