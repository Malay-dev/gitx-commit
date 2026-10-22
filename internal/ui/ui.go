package ui

import (
	"fmt"
	"os"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// isTerminal returns true if stderr is a terminal (for color support).
func isTerminal() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func colorize(color, msg string) string {
	if !isTerminal() {
		return msg
	}
	return color + msg + colorReset
}

// Success prints a green success message.
func Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, colorize(colorGreen, "✓ "+msg))
}

// Error prints a red error message.
func Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, colorize(colorRed, "✗ "+msg))
}

// Warn prints a yellow warning message.
func Warn(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, colorize(colorYellow, "⚠ "+msg))
}

// Info prints a cyan informational message.
func Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, colorize(colorCyan, "ℹ "+msg))
}

// Field prints a labeled value in a consistent format.
func Field(label, value string) {
	fmt.Fprintf(os.Stderr, "  %-18s %s\n", colorize(colorBold, label+":"), value)
}
