package utils

import (
	"fmt"
	"strings"
)

// TruncateID shortens a UUID or other identifier for display purposes.
// It shows the first 8 characters followed by "...".
// If preferShort is true and a short name is provided, it will use that instead.
func TruncateID(id string, shortName string, preferShort bool) string {
	if preferShort && shortName != "" {
		return shortName
	}
	
	if len(id) > 8 {
		return id[:8] + "..."
	}
	return id
}

// FormatResourceName formats a resource name and ID for display.
// If the ID is empty, it returns just the name.
// Otherwise, it returns "name (id)" with the ID truncated.
func FormatResourceName(name, id string) string {
	if id == "" {
		return name
	}
	
	truncID := TruncateID(id, "", false)
	return fmt.Sprintf("%s (%s)", name, truncID)
}

// FormatSuccess formats a success message with an emoji.
func FormatSuccess(message string) string {
	return fmt.Sprintf("✅ %s", message)
}

// FormatInfo formats an informational message.
func FormatInfo(message string) string {
	return message
}

// FormatWarning formats a warning message with an emoji.
func FormatWarning(message string) string {
	return fmt.Sprintf("⚠️ %s", message)
}

// FormatError formats an error message with an emoji.
func FormatError(message string) string {
	return fmt.Sprintf("❌ %s", message)
}

// FormatAction formats an action message with an emoji.
func FormatAction(message string) string {
	return fmt.Sprintf("🔨 %s", message)
}

// TruncateOutput truncates a command output for display.
// It keeps the first maxLines lines.
func TruncateOutput(output string, maxLines int) string {
	lines := strings.Split(output, "\n")
	if len(lines) <= maxLines {
		return output
	}
	
	truncated := lines[:maxLines]
	return strings.Join(truncated, "\n") + fmt.Sprintf("\n... (%d more lines truncated)", len(lines)-maxLines)
}