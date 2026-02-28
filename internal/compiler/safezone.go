package compiler

import (
	"fmt"
	"strings"
)

// Safe zone markers for recompilation.
const (
	GeneratedStartMarker = "AGENTSPEC GENERATED START"
	GeneratedEndMarker   = "AGENTSPEC GENERATED END"
	UserCodeStartMarker  = "USER CODE START"
	UserCodeEndMarker    = "USER CODE END"
)

// SafeZone represents a section of a file that is either generated or user-written.
type SafeZone struct {
	Type    string // "generated" or "user"
	Content string
}

// ParseSafeZones parses a file's content into safe zones, separating generated
// and user-written sections based on markers.
func ParseSafeZones(content, commentPrefix string) []SafeZone {
	lines := strings.Split(content, "\n")
	var zones []SafeZone
	var current strings.Builder
	currentType := "" // "", "generated", "user", or "outside" for non-marker content

	genStart := fmt.Sprintf("%s --- %s ---", commentPrefix, GeneratedStartMarker)
	genEnd := fmt.Sprintf("%s --- %s ---", commentPrefix, GeneratedEndMarker)
	userStart := fmt.Sprintf("%s --- %s ---", commentPrefix, UserCodeStartMarker)
	userEnd := fmt.Sprintf("%s --- %s ---", commentPrefix, UserCodeEndMarker)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == genStart:
			if currentType != "" && current.Len() > 0 {
				zones = append(zones, SafeZone{Type: currentType, Content: current.String()})
				current.Reset()
			}
			currentType = "generated"
			current.WriteString(line)
			current.WriteString("\n")

		case trimmed == genEnd:
			current.WriteString(line)
			current.WriteString("\n")
			zones = append(zones, SafeZone{Type: "generated", Content: current.String()})
			current.Reset()
			currentType = ""

		case trimmed == userStart:
			if currentType != "" && current.Len() > 0 {
				zones = append(zones, SafeZone{Type: currentType, Content: current.String()})
				current.Reset()
			}
			currentType = "user"
			current.WriteString(line)
			current.WriteString("\n")

		case trimmed == userEnd:
			current.WriteString(line)
			current.WriteString("\n")
			zones = append(zones, SafeZone{Type: "user", Content: current.String()})
			current.Reset()
			currentType = ""

		default:
			if currentType == "" {
				currentType = "outside"
			}
			current.WriteString(line)
			current.WriteString("\n")
		}
	}

	if current.Len() > 0 {
		if currentType == "" {
			currentType = "outside"
		}
		zones = append(zones, SafeZone{Type: currentType, Content: current.String()})
	}

	return zones
}

// ExtractUserCode extracts user-written code sections from an existing file.
// Returns a map of section index → user code content.
func ExtractUserCode(content, commentPrefix string) map[int]string {
	zones := ParseSafeZones(content, commentPrefix)
	userSections := make(map[int]string)
	idx := 0
	for _, z := range zones {
		if z.Type == "user" {
			userSections[idx] = z.Content
			idx++
		}
	}
	return userSections
}

// MergeWithUserCode merges new generated content with preserved user code sections.
// The newContent should contain both generated and user code markers. User code
// sections from existingUserCode are injected into the corresponding USER CODE zones.
func MergeWithUserCode(newContent, commentPrefix string, existingUserCode map[int]string) string {
	if len(existingUserCode) == 0 {
		return newContent
	}

	zones := ParseSafeZones(newContent, commentPrefix)
	var result strings.Builder
	userIdx := 0

	for _, z := range zones {
		if z.Type == "user" {
			if preserved, ok := existingUserCode[userIdx]; ok {
				result.WriteString(preserved)
			} else {
				result.WriteString(z.Content)
			}
			userIdx++
		} else {
			result.WriteString(z.Content)
		}
	}

	return result.String()
}

// WrapGenerated wraps content with generated markers.
func WrapGenerated(content, commentPrefix string) string {
	return fmt.Sprintf(
		"%s --- %s ---\n%s Do not edit between these markers; changes will be overwritten on recompile\n%s%s --- %s ---\n",
		commentPrefix, GeneratedStartMarker,
		commentPrefix,
		content,
		commentPrefix, GeneratedEndMarker,
	)
}

// WrapUserCode wraps a placeholder with user code markers.
func WrapUserCode(placeholder, commentPrefix string) string {
	return fmt.Sprintf(
		"%s --- %s ---\n%s Your custom code here is preserved across recompilations\n%s%s --- %s ---\n",
		commentPrefix, UserCodeStartMarker,
		commentPrefix,
		placeholder,
		commentPrefix, UserCodeEndMarker,
	)
}

// CommentPrefixForLanguage returns the line comment prefix for a given language.
func CommentPrefixForLanguage(lang string) string {
	switch strings.ToLower(lang) {
	case "python", "py", "yaml", "yml", "bash", "sh":
		return "#"
	case "javascript", "js", "typescript", "ts", "go", "java", "c", "cpp", "rust":
		return "//"
	default:
		return "#"
	}
}
