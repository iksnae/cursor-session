package internal

import (
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name string
		uuid string
		want bool
	}{
		{"valid lowercase UUID", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid uppercase UUID", "550E8400-E29B-41D4-A716-446655440000", true},
		{"valid mixed case UUID", "550E8400-e29b-41D4-A716-446655440000", true},
		{"invalid - too short", "550e8400-e29b-41d4-a716", false},
		{"invalid - missing dashes", "550e8400e29b41d4a716446655440000", false},
		{"invalid - wrong format", "not-a-uuid", false},
		{"empty string", "", false},
		{"valid with x", "550e8400-e29b-41d4-a716-44665544000x", false}, // x is not valid hex
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidUUID(tt.uuid)
			if got != tt.want {
				t.Errorf("isValidUUID(%q) = %v, want %v", tt.uuid, got, tt.want)
			}
		})
	}
}

func TestIsReadableText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"readable text", "Hello, world!", true},
		{"readable with newlines", "Hello\nWorld\n", true},
		{"readable with tabs", "Hello\tWorld", true},
		{"empty string", "", false},
		{"binary data", string([]byte{0x00, 0x01, 0x02, 0x03}), false},
		{"mostly binary", "Hello" + string([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}), false},
		{"short readable", "Hi", true},
		{"short with control chars", string([]byte{'H', 0x01, 'i'}), false},
		{"unicode text", "Hello 世界", true},
		{"invalid UTF-8", string([]byte{0xff, 0xfe, 0xfd}), false},
		{"70% printable", "Hello" + string([]byte{0x00, 0x01, 0x02}), false},                                                // 5 printable, 3 non-printable = 62.5%, but < 5 chars so all must be printable
		{"long text with some binary", "This is a long readable text with some content" + string([]byte{0x00, 0x01}), true}, // > 70% printable
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isReadableText(tt.text)
			if got != tt.want {
				t.Errorf("isReadableText(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestIsHashLike(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"valid hash 32 chars", "a1b2c3d4e5f6789012345678901234ab", true},
		{"valid hash 64 chars", "a1b2c3d4e5f6789012345678901234aba1b2c3d4e5f6789012345678901234ab", true},
		{"valid hash 16 chars", "a1b2c3d4e5f67890", true},
		{"valid hash 128 chars", "a1b2c3d4e5f6789012345678901234aba1b2c3d4e5f6789012345678901234aba1b2c3d4e5f6789012345678901234aba1b2c3d4e5f6789012345678901234ab", true},
		{"too short", "a1b2c3d4", false},
		{"too long", "a1b2c3d4e5f6789012345678901234aba1b2c3d4e5f6789012345678901234aba1b2c3d4e5f6789012345678901234aba1b2c3d4e5f6789012345678901234abx", false},
		{"contains non-hex", "a1b2c3d4e5f6789012345678901234gx", false},
		{"contains uppercase", "A1B2C3D4E5F6789012345678901234AB", true},
		{"mixed case", "a1B2c3D4e5F6789012345678901234Ab", true},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHashLike(tt.s)
			if got != tt.want {
				t.Errorf("isHashLike(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestTryBase64Decode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantLen int
	}{
		{"valid base64", base64.StdEncoding.EncodeToString([]byte("Hello, world!")), false, 13},
		{"valid URL-safe base64", base64.URLEncoding.EncodeToString([]byte("Hello, world!")), false, 13},
		{"base64 without padding", base64.StdEncoding.EncodeToString([]byte("test"))[:len(base64.StdEncoding.EncodeToString([]byte("test")))-1], false, 4},
		{"invalid base64", "not base64!", true, 0},
		{"empty string", "", false, 0}, // Empty string decodes to empty bytes, not an error
		{"binary data", string([]byte{0xff, 0xfe, 0xfd}), true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tryBase64Decode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("tryBase64Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("tryBase64Decode() length = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestTryHexDecode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantLen int
	}{
		{"valid hex", hex.EncodeToString([]byte("Hello")), false, 5},
		{"valid hex with spaces", "48 65 6c 6c 6f", false, 5},
		{"valid hex with newlines", "48\n65\n6c\n6c\n6f", false, 5},
		{"valid hex with tabs", "48\t65\t6c\t6c\t6f", false, 5},
		{"valid hex uppercase", "48656C6C6F", false, 5},
		{"invalid hex", "not hex!", true, 0},
		{"odd length hex", "48656c6c6", true, 0},
		{"empty string", "", false, 0},
		{"hex with mixed case", "48e56C6c6F", false, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tryHexDecode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("tryHexDecode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("tryHexDecode() length = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestExtractJSONFromBinary(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantFound bool
		wantJSON  string
	}{
		{"valid JSON in binary", []byte("prefix{\"key\":\"value\"}suffix"), true, "{\"key\":\"value\"}"},
		{"JSON at start", []byte("{\"key\":\"value\"}suffix"), true, "{\"key\":\"value\"}"},
		{"JSON at end", []byte("prefix{\"key\":\"value\"}"), true, "{\"key\":\"value\"}"},
		{"nested JSON", []byte("prefix{\"key\":{\"nested\":\"value\"}}suffix"), true, "{\"key\":{\"nested\":\"value\"}}"},
		{"JSON with escaped quotes", []byte("prefix{\"key\":\"value\\\"with\\\"quotes\"}suffix"), true, "{\"key\":\"value\\\"with\\\"quotes\"}"},
		{"no JSON", []byte("just binary data"), false, ""},
		{"invalid JSON - unclosed brace", []byte("prefix{\"key\":\"value\""), false, ""},
		{"invalid JSON - invalid UTF-8", []byte("prefix{\"key\":\"" + string([]byte{0xff, 0xfe}) + "\"}suffix"), false, ""},
		{"empty data", []byte(""), false, ""},
		{"only opening brace", []byte("{"), false, ""},
		{"JSON with array", []byte("prefix{\"key\":[\"value1\",\"value2\"]}suffix"), true, "{\"key\":[\"value1\",\"value2\"]}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := extractJSONFromBinary(tt.data)
			if found != tt.wantFound {
				t.Errorf("extractJSONFromBinary() found = %v, want %v", found, tt.wantFound)
				return
			}
			if tt.wantFound {
				if string(got) != tt.wantJSON {
					t.Errorf("extractJSONFromBinary() = %q, want %q", string(got), tt.wantJSON)
				}
			}
		})
	}
}

func TestParseTextMessageFormat(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		sessionID string
		want      bool // whether bubble should be created
		wantText  string
		wantType  int
	}{
		{"valid format", "key123", "hello$550e8400-e29b-41d4-a716-446655440000", "session1", true, "hello", 1},
		{"valid without UUID", "key123", "hello$", "session1", true, "hello", 1},
		{"valid with spaces", "key123", "  hello world  $550e8400-e29b-41d4-a716-446655440000", "session1", true, "hello world", 1},
		{"valid with quotes", "key123", "\"hello\"$550e8400-e29b-41d4-a716-446655440000", "session1", true, "hello", 1},
		{"valid with single quotes", "key123", "'hello'$550e8400-e29b-41d4-a716-446655440000", "session1", true, "hello", 1},
		{"no dollar sign", "key123", "hello world", "session1", false, "", 0},
		{"dollar at start", "key123", "$550e8400-e29b-41d4-a716-446655440000", "session1", false, "", 0},
		{"empty text", "key123", "$550e8400-e29b-41d4-a716-446655440000", "session1", false, "", 0},
		{"binary data", "key123", string([]byte{0x00, 0x01, 0x02, '$', 'u', 'u', 'i', 'd'}), "session1", false, "", 0},
		{"invalid UTF-8", "key123", string([]byte{0xff, 0xfe, '$', 'u', 'u', 'i', 'd'}), "session1", false, "", 0},
		{"with control chars", "key123", "hello\n\t$550e8400-e29b-41d4-a716-446655440000", "session1", true, "hello", 1},
		{"short key with valid UUID", "key", "hello$550e8400-e29b-41d4-a716-446655440000", "session1", true, "hello", 1}, // Key length doesn't matter for parsing
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTextMessageFormat(tt.key, tt.value, tt.sessionID)
			if (got != nil) != tt.want {
				t.Errorf("parseTextMessageFormat() returned bubble = %v, want %v", got != nil, tt.want)
				return
			}
			if tt.want {
				if got.Text != tt.wantText {
					t.Errorf("parseTextMessageFormat() Text = %q, want %q", got.Text, tt.wantText)
				}
				if got.Type != tt.wantType {
					t.Errorf("parseTextMessageFormat() Type = %d, want %d", got.Type, tt.wantType)
				}
				if got.ChatID != tt.sessionID {
					t.Errorf("parseTextMessageFormat() ChatID = %q, want %q", got.ChatID, tt.sessionID)
				}
			}
		})
	}
}

func TestParseMessageToBubble(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		id        string
		role      string
		data      map[string]interface{}
		sessionID string
		wantErr   bool
		wantType  int
		wantText  string
	}{
		{
			name:      "user message with text content",
			key:       "key12345678",
			id:        "msg1",
			role:      "user",
			data:      map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "text", "text": "Hello"}}},
			sessionID: "session1",
			wantErr:   false,
			wantType:  1,
			wantText:  "Hello",
		},
		{
			name:      "assistant message",
			key:       "key12345678",
			id:        "msg2",
			role:      "assistant",
			data:      map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "text", "text": "Hi there"}}},
			sessionID: "session1",
			wantErr:   false,
			wantType:  2,
			wantText:  "Hi there",
		},
		{
			name: "message with tool call",
			key:  "key12345678",
			id:   "msg3",
			role: "assistant",
			data: map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type":         "tool_call",
						"name":         "read_file",
						"tool_call_id": "call1",
						"arguments":    `{"path": "file.txt"}`,
					},
				},
			},
			sessionID: "session1",
			wantErr:   false,
			wantType:  2,
			wantText:  "[Tool Call]\nTool: read_file\nID: call1\nArguments: {\"path\": \"file.txt\"}",
		},
		{
			name: "message with timestamp",
			key:  "key12345678",
			id:   "msg4",
			role: "user",
			data: map[string]interface{}{
				"content":   []interface{}{map[string]interface{}{"type": "text", "text": "Hello"}},
				"timestamp": float64(1000000),
			},
			sessionID: "session1",
			wantErr:   false,
			wantType:  1,
			wantText:  "Hello",
		},
		{
			name:      "unknown role defaults to assistant",
			key:       "key12345678",
			id:        "msg5",
			role:      "unknown",
			data:      map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "text", "text": "Hello"}}},
			sessionID: "session1",
			wantErr:   false,
			wantType:  2,
			wantText:  "Hello",
		},
		{
			name:      "short key",
			key:       "key",
			id:        "msg6",
			role:      "user",
			data:      map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "text", "text": "Hello"}}},
			sessionID: "session1",
			wantErr:   false,
			wantType:  1,
			wantText:  "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMessageToBubble(tt.key, tt.id, tt.role, tt.data, tt.sessionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMessageToBubble() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Type != tt.wantType {
					t.Errorf("parseMessageToBubble() Type = %d, want %d", got.Type, tt.wantType)
				}
				if got.Text != tt.wantText {
					t.Errorf("parseMessageToBubble() Text = %q, want %q", got.Text, tt.wantText)
				}
				if got.ChatID != tt.sessionID {
					t.Errorf("parseMessageToBubble() ChatID = %q, want %q", got.ChatID, tt.sessionID)
				}
			}
		})
	}
}
