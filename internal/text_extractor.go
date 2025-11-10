package internal

import (
	"fmt"
	"regexp"
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
		// Check if text contains old format redacted reasoning and reformat it
		text := reformatRedactedReasoning(bubble.Text)
		if text != "" {
			textParts = append(textParts, text)
		}
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

// reformatRedactedReasoning detects and reformats old-style redacted reasoning
// Old format: [Redacted Reasoning: hash]
// New format: ```\n[Redacted Reasoning]\nhash\n``` (or decoded if possible)
// If the text only contains redacted reasoning, it returns empty string (don't display)
func reformatRedactedReasoning(text string) string {
	// Pattern: [Redacted Reasoning: ...]
	pattern := `\[Redacted Reasoning:\s*([^\]]+)\]`

	// Check if the entire text is just redacted reasoning
	matched, _ := regexp.MatchString(`^\s*`+pattern+`\s*$`, text)
	if matched {
		// Extract the hash/reasoning content
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			encoded := strings.TrimSpace(matches[1])
			// Try to decode the redacted reasoning
			decoded, wasDecoded := decodeRedactedReasoning(encoded)
			if wasDecoded {
				return fmt.Sprintf("```\n[Redacted Reasoning - Decoded]\n%s\n```", decoded)
			}
			// Format as code block with original encoded value
			return fmt.Sprintf("```\n[Redacted Reasoning]\n%s\n```", encoded)
		}
		// If we can't extract, return empty (don't display)
		return ""
	}

	// If text contains redacted reasoning but has other content, replace it with code block format
	re := regexp.MustCompile(pattern)
	text = re.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the content after the colon
		parts := strings.SplitN(match, ":", 2)
		if len(parts) == 2 {
			encoded := strings.TrimSpace(strings.TrimSuffix(parts[1], "]"))
			// Try to decode
			decoded, wasDecoded := decodeRedactedReasoning(encoded)
			if wasDecoded {
				return fmt.Sprintf("```\n[Redacted Reasoning - Decoded]\n%s\n```", decoded)
			}
			return fmt.Sprintf("```\n[Redacted Reasoning]\n%s\n```", encoded)
		}
		return match
	})

	return text
}
