package internal

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RichTextNode represents a node in the rich text structure
type RichTextNode struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	Content  string         `json:"content,omitempty"` // Some nodes may have content field
	Value    string         `json:"value,omitempty"`   // Some nodes may have value field
	Children []RichTextNode `json:"children,omitempty"`
}

// RichTextRoot represents the root of the rich text structure
type RichTextRoot struct {
	Root RichTextNode `json:"root"`
}

// ExtractTextFromRichText parses richText JSON and extracts plain text
// Based on cursor-chat-browser implementation
func ExtractTextFromRichText(richTextJSON string) (string, error) {
	if richTextJSON == "" {
		return "", nil
	}

	// Parse the JSON - it might be a root object with root.children
	var richTextData map[string]interface{}
	if err := json.Unmarshal([]byte(richTextJSON), &richTextData); err != nil {
		return "", fmt.Errorf("failed to parse richText JSON: %w", err)
	}

	// Check for root.children structure (most common)
	if root, ok := richTextData["root"].(map[string]interface{}); ok {
		if children, ok := root["children"].([]interface{}); ok {
			result := extractTextFromChildrenInterface(children)
			if result != "" {
				return result, nil
			}
		}
	}

	// Try direct children array
	if children, ok := richTextData["children"].([]interface{}); ok {
		result := extractTextFromChildrenInterface(children)
		if result != "" {
			return result, nil
		}
	}

	// Try parsing as RichTextRoot (structured)
	var rootStruct RichTextRoot
	if err := json.Unmarshal([]byte(richTextJSON), &rootStruct); err == nil {
		result := extractTextFromNode(rootStruct.Root)
		if result != "" {
			return result, nil
		}
	}

	// Try parsing as a direct node
	var node RichTextNode
	if err := json.Unmarshal([]byte(richTextJSON), &node); err == nil {
		result := extractTextFromNode(node)
		if result != "" {
			return result, nil
		}
	}

	// Try parsing as an array of nodes
	var nodes []RichTextNode
	if err := json.Unmarshal([]byte(richTextJSON), &nodes); err == nil {
		result := extractTextFromChildren(nodes)
		if result != "" {
			return result, nil
		}
	}

	// If all structured parsing fails, return error
	return "", fmt.Errorf("failed to parse richText JSON in any known format")
}

// extractTextFromChildrenInterface extracts text from an array of interface{} (from JSON parsing)
func extractTextFromChildrenInterface(children []interface{}) string {
	var text string
	for _, childInterface := range children {
		childMap, ok := childInterface.(map[string]interface{})
		if !ok {
			continue
		}

		childType, _ := childMap["type"].(string)
		childText, _ := childMap["text"].(string)

		if childType == "text" && childText != "" {
			text += childText
		} else if childType == "code" {
			// For code blocks, add markdown code fences
			if childChildren, ok := childMap["children"].([]interface{}); ok {
				codeText := extractTextFromChildrenInterface(childChildren)
				if codeText != "" {
					text += "\n```\n" + codeText + "\n```\n"
				}
			}
		} else if childType == "redacted_reasoning" || childType == "redacted-reasoning" {
			// Extract redacted reasoning and format in code block
			var reasoningText string
			if childChildren, ok := childMap["children"].([]interface{}); ok {
				reasoningText = extractTextFromChildrenInterface(childChildren)
			}
			if reasoningText == "" {
				// Try to get from content or data fields
				if content, ok := childMap["content"].(string); ok {
					reasoningText = content
				} else if data, ok := childMap["data"].(string); ok {
					reasoningText = data
				}
			}
			if reasoningText != "" {
				// Try to decode the redacted reasoning (may be base64url + protobuf encoded)
				decoded, wasDecoded := decodeRedactedReasoning(reasoningText)
				if wasDecoded {
					// Successfully decoded - show the actual reasoning content
					text += fmt.Sprintf("\n```\n[Redacted Reasoning - Decoded]\n%s\n```\n", decoded)
				} else {
					// Could not decode - show as-is in code block
					text += fmt.Sprintf("\n```\n[Redacted Reasoning]\n%s\n```\n", reasoningText)
				}
			}
		} else {
			// Recursively process children
			if childChildren, ok := childMap["children"].([]interface{}); ok {
				text += extractTextFromChildrenInterface(childChildren)
			}
		}
	}
	return text
}

// extractTextFromNode recursively extracts text from a node
func extractTextFromNode(node RichTextNode) string {
	var text string

	// Handle different node types
	switch node.Type {
	case "text":
		if node.Text != "" {
			text += node.Text
		}
	case "code":
		// For code blocks, add markdown code fences
		codeText := extractTextFromChildren(node.Children)
		if codeText != "" {
			text += "\n```\n" + codeText + "\n```\n"
		}
	case "thinking", "tool", "tool_call", "function_call":
		// Extract thinking/tool call content
		thinkingText := extractTextFromChildren(node.Children)
		if thinkingText != "" {
			text += fmt.Sprintf("\n[%s]\n%s\n", node.Type, thinkingText)
		}
	case "redacted_reasoning", "redacted-reasoning":
		// Extract redacted reasoning and format in code block
		reasoningText := extractTextFromChildren(node.Children)
		if reasoningText == "" {
			// Try to get from content or value fields
			if node.Content != "" {
				reasoningText = node.Content
			} else if node.Value != "" {
				reasoningText = node.Value
			}
		}
		if reasoningText != "" {
			// Try to decode the redacted reasoning (may be base64url + protobuf encoded)
			decoded, wasDecoded := decodeRedactedReasoning(reasoningText)
			if wasDecoded {
				// Successfully decoded - show the actual reasoning content
				text += fmt.Sprintf("\n```\n[Redacted Reasoning - Decoded]\n%s\n```\n", decoded)
			} else {
				// Could not decode - show as-is in code block
				text += fmt.Sprintf("\n```\n[Redacted Reasoning]\n%s\n```\n", reasoningText)
			}
		}
	default:
		// For unknown types, try to extract any text content from various fields
		if node.Text != "" {
			text += node.Text
		}
		if node.Content != "" {
			if text != "" {
				text += "\n"
			}
			text += node.Content
		}
		if node.Value != "" {
			if text != "" {
				text += "\n"
			}
			text += node.Value
		}
	}

	// Recursively process children (always do this to catch nested content)
	if len(node.Children) > 0 {
		childrenText := extractTextFromChildren(node.Children)
		if childrenText != "" {
			if text != "" && !strings.HasSuffix(text, "\n") {
				text += "\n"
			}
			text += childrenText
		}
	}

	return text
}

// extractTextFromChildren extracts text from an array of nodes
func extractTextFromChildren(children []RichTextNode) string {
	var text string
	for _, child := range children {
		text += extractTextFromNode(child)
	}
	return text
}
