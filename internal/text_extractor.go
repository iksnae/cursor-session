package internal

import (
	"fmt"
	"strings"
)

// ExtractTextFromBubble extracts text from a bubble using three-tier strategy:
// 1. Primary: Use bubble.text if available
// 2. Fallback: Parse bubble.richText JSON structure (including thinking/tool calls)
// 3. Enhancement: Append bubble.codeBlocks[] as markdown code fences
func ExtractTextFromBubble(bubble *RawBubble) (string, error) {
	var textParts []string

	// Tier 1: Primary text field
	if bubble.Text != "" {
		textParts = append(textParts, bubble.Text)
	}

	// Tier 2: Parse richText JSON structure (even if text exists, richText may have additional info)
	if bubble.RichText != "" {
		richText, err := ExtractTextFromRichText(bubble.RichText)
		if err != nil {
			// If richText parsing fails, try fallback extraction
			LogDebug("Failed to parse richText JSON: %v, trying fallback extraction", err)
			richText = extractFallbackText(bubble.RichText)
			// Also try to extract from raw JSON structure more aggressively
			if richText == "" {
				richText = extractFromRawJSON(bubble.RichText)
			}
		}
		if richText != "" {
			// Only add if it's different from the primary text to avoid duplication
			if bubble.Text == "" || !strings.Contains(bubble.Text, richText) {
				textParts = append(textParts, richText)
			}
		}
	}

	// Tier 3: Append code blocks
	if len(bubble.CodeBlocks) > 0 {
		for _, codeBlock := range bubble.CodeBlocks {
			if codeBlock.Content != "" {
				lang := codeBlock.Language
				if lang == "" {
					lang = ""
				}
				textParts = append(textParts, fmt.Sprintf("```%s\n%s\n```", lang, codeBlock.Content))
			}
		}
	}

	// Combine all parts
	result := strings.Join(textParts, "\n\n")

	// If we still have no text, return a placeholder to indicate the message exists
	if result == "" {
		return "[Message with no extractable text content]", nil
	}

	return result, nil
}

// extractFallbackText tries to extract any readable text from a JSON string
// This is a last resort when proper parsing fails
func extractFallbackText(jsonStr string) string {
	// Try to find text fields in the JSON
	// This is a simple heuristic - look for "text":"..." patterns
	var result strings.Builder
	escapeNext := false

	for i := 0; i < len(jsonStr)-6; i++ {
		if escapeNext {
			escapeNext = false
			continue
		}

		if jsonStr[i] == '\\' {
			escapeNext = true
			continue
		}

		// Look for "text": pattern
		if i+6 < len(jsonStr) && jsonStr[i:i+6] == `"text"` {
			// Skip to the value
			j := i + 6
			for j < len(jsonStr) && (jsonStr[j] == ' ' || jsonStr[j] == ':') {
				j++
			}
			if j < len(jsonStr) && jsonStr[j] == '"' {
				// Extract the string value
				j++
				start := j
				for j < len(jsonStr) && (jsonStr[j] != '"' || escapeNext) {
					if jsonStr[j] == '\\' {
						escapeNext = true
					} else {
						escapeNext = false
					}
					j++
				}
				if j < len(jsonStr) {
					value := jsonStr[start:j]
					if value != "" {
						result.WriteString(value)
						result.WriteString(" ")
					}
				}
			}
		}
	}

	return strings.TrimSpace(result.String())
}

// extractFromRawJSON tries to extract text from raw JSON using more aggressive parsing
func extractFromRawJSON(jsonStr string) string {
	// Try to find common patterns in JSON that might contain text
	// Look for: "content", "value", "message", "text", "thinking", "tool"
	patterns := []string{`"content"`, `"value"`, `"message"`, `"text"`, `"thinking"`, `"tool"`, `"name"`, `"description"`}
	var result strings.Builder

	for _, pattern := range patterns {
		idx := strings.Index(jsonStr, pattern)
		if idx == -1 {
			continue
		}

		// Skip to the value after the pattern
		start := idx + len(pattern)
		for start < len(jsonStr) && (jsonStr[start] == ' ' || jsonStr[start] == ':') {
			start++
		}

		if start >= len(jsonStr) {
			continue
		}

		// Extract string value
		if jsonStr[start] == '"' {
			start++
			end := start
			escapeNext := false
			for end < len(jsonStr) && (jsonStr[end] != '"' || escapeNext) {
				if jsonStr[end] == '\\' {
					escapeNext = true
				} else {
					escapeNext = false
				}
				end++
			}
			if end < len(jsonStr) {
				value := jsonStr[start:end]
				// Unescape basic JSON escapes
				value = strings.ReplaceAll(value, `\"`, `"`)
				value = strings.ReplaceAll(value, `\\`, `\`)
				value = strings.ReplaceAll(value, `\n`, "\n")
				value = strings.ReplaceAll(value, `\t`, "\t")
				if len(value) > 10 { // Only add substantial content
					result.WriteString(value)
					result.WriteString("\n")
				}
			}
		}
	}

	return strings.TrimSpace(result.String())
}
