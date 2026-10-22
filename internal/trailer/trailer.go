// Package trailer handles adding git trailers (Assisted-by, Commit-Mode, etc.)
// to commit messages. Trailers are key-value metadata lines in the last
// paragraph of a commit message.
//
// Example commit message with trailers:
//
//	feat(auth): add OAuth login
//
//	Implemented OAuth2 flow with Google provider.
//
//	Signed-off-by: Malay Kumar
//	Assisted-by: Kiro/2.32.0
package trailer

import (
	"strings"
	"unicode"
)

// FormatTrailer returns a single trailer line: "Key: Value"
func FormatTrailer(key, value string) string {
	return key + ": " + value
}

// BuildTrailerValue constructs the value part of a trailer from
// detection results and config settings.
//
// Examples:
//
//	BuildTrailerValue("Kiro", "claude-opus-4.6", true)   → "Kiro (claude-opus-4.6)"
//	BuildTrailerValue("Kiro", "claude-opus-4.6", false)  → "Kiro"
//	BuildTrailerValue("Cursor", "", true)                → "Cursor"
func BuildTrailerValue(tool, model string, includeModel bool) string {
	if includeModel && model != "" {
		return tool + " (" + model + ")"
	}
	return tool
}

// HasTrailer checks if the message already contains a trailer with the given key.
// This prevents duplicate trailers when hooks run multiple times (idempotency).
func HasTrailer(message, key string) bool {
	// Look for "Key:" at the start of any line (case-sensitive)
	prefix := key + ":"
	for _, line := range strings.Split(message, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return true
		}
	}
	return false
}

// AddTrailer appends a "Key: Value" trailer to the commit message.
//
// Rules:
//   - If the message is empty, return it unchanged
//   - If the trailer already exists (same key), return unchanged (idempotent)
//   - If the message already has trailers (last paragraph has "Key: Value" lines),
//     append on the next line (no blank line between trailers)
//   - If the message has no trailers, add a blank line separator first
//
// The returned message always ends with a newline.
func AddTrailer(message, key, value string) string {
	if strings.TrimSpace(message) == "" {
		return message
	}

	// Don't add duplicate trailers
	if HasTrailer(message, key) {
		return message
	}

	trailerLine := FormatTrailer(key, value)

	// Remove trailing whitespace/newlines for clean manipulation
	trimmed := strings.TrimRight(message, " \t\n")

	// Split into paragraphs (separated by double newlines)
	// The last paragraph might be existing trailers
	if hasTrailerBlock(trimmed) {
		// Last paragraph is trailers — append directly (no blank line)
		return trimmed + "\n" + trailerLine + "\n"
	}

	// No existing trailers — add blank line separator then trailer
	return trimmed + "\n\n" + trailerLine + "\n"
}

// hasTrailerBlock checks if the last paragraph of the message looks like
// a trailer block. A trailer line matches the pattern "Key: Value" where
// Key starts with a letter and contains no spaces before the colon.
//
// Examples of trailer lines:
//
//	Signed-off-by: Malay Kumar     → yes
//	Assisted-by: Kiro/2.32.0      → yes
//	Co-authored-by: Claude         → yes
//
// Examples of non-trailer lines:
//
//	This is a normal sentence.     → no (no colon pattern)
//	Note: this looks like one      → yes (false positive, but acceptable)
func hasTrailerBlock(message string) bool {
	// Find the last paragraph
	paragraphs := strings.Split(message, "\n\n")
	lastParagraph := paragraphs[len(paragraphs)-1]

	// Check each line in the last paragraph
	lines := strings.Split(strings.TrimSpace(lastParagraph), "\n")
	if len(lines) == 0 {
		return false
	}

	// The last paragraph is a trailer block if ALL its lines look like trailers
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !looksLikeTrailer(line) {
			return false
		}
	}
	return true
}

// looksLikeTrailer checks if a line matches "Word-Word: value" pattern.
// The key part (before colon) must:
//   - Start with a letter
//   - Contain no spaces (hyphens are OK: "Co-authored-by")
//   - Be followed by ": " (colon + space)
//   - Key must be at least 2 characters (avoids matching "feat: add login")
func looksLikeTrailer(line string) bool {
	colonIdx := strings.Index(line, ": ")
	if colonIdx <= 0 {
		return false
	}

	key := line[:colonIdx]

	// Key must start with a letter
	if len(key) == 0 || !unicode.IsLetter(rune(key[0])) {
		return false
	}

	// Key must not contain spaces
	if strings.Contains(key, " ") {
		return false
	}

	// Key must contain a hyphen or be a known single-word trailer
	// This avoids matching "feat: add login" as a trailer
	// Common single-word trailers: Refs, Fixes, Closes, Resolves
	knownSingleWord := map[string]bool{
		"Refs": true, "Fixes": true, "Closes": true, "Resolves": true,
	}
	if !strings.Contains(key, "-") && !knownSingleWord[key] {
		return false
	}

	return true
}
