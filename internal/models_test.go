package internal

import (
	"testing"
)

func TestParseRawBubble(t *testing.T) {
	key := "bubbleId:chat123:bubble456"
	value := `{"text":"Hello","timestamp":1000,"type":1}`

	bubble, err := ParseRawBubble(key, value)
	if err != nil {
		t.Fatalf("ParseRawBubble() error = %v", err)
	}

	if bubble.ChatID != "chat123" {
		t.Errorf("ChatID = %v, want chat123", bubble.ChatID)
	}

	if bubble.BubbleID != "bubble456" {
		t.Errorf("BubbleID = %v, want bubble456", bubble.BubbleID)
	}

	if bubble.Text != "Hello" {
		t.Errorf("Text = %v, want Hello", bubble.Text)
	}
}

func TestParseRawComposer(t *testing.T) {
	key := "composerData:composer123"
	value := `{"name":"Test Conversation","createdAt":1000}`

	composer, err := ParseRawComposer(key, value)
	if err != nil {
		t.Fatalf("ParseRawComposer() error = %v", err)
	}

	if composer.ComposerID != "composer123" {
		t.Errorf("ComposerID = %v, want composer123", composer.ComposerID)
	}

	if composer.Name != "Test Conversation" {
		t.Errorf("Name = %v, want Test Conversation", composer.Name)
	}
}

func TestParseRawBubble_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{
			name:    "invalid key format - missing parts",
			key:     "bubbleId:chat1",
			value:   `{"text":"Hello"}`,
			wantErr: true,
		},
		{
			name:    "invalid key format - wrong prefix",
			key:     "wrong:chat1:bubble1",
			value:   `{"text":"Hello"}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			key:     "bubbleId:chat1:bubble1",
			value:   `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "empty value",
			key:     "bubbleId:chat1:bubble1",
			value:   ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRawBubble(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRawBubble() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseRawComposer_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{
			name:    "invalid key format",
			key:     "composerData",
			value:   `{"name":"Test"}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			key:     "composerData:composer1",
			value:   `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "empty value",
			key:     "composerData:composer1",
			value:   ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRawComposer(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRawComposer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseMessageContext(t *testing.T) {
	key := "messageRequestContext:composer1:context1"
	value := `{"bubbleId":"bubble1","composerId":"composer1","contextId":"context1"}`

	context, err := ParseMessageContext(key, value)
	if err != nil {
		t.Fatalf("ParseMessageContext() error = %v", err)
	}

	if context.ComposerID != "composer1" {
		t.Errorf("ComposerID = %v, want composer1", context.ComposerID)
	}

	if context.ContextID != "context1" {
		t.Errorf("ContextID = %v, want context1", context.ContextID)
	}
}

func TestParseMessageContext_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{
			name:    "invalid key format - missing parts",
			key:     "messageRequestContext:composer1",
			value:   `{"bubbleId":"bubble1"}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			key:     "messageRequestContext:composer1:context1",
			value:   `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseMessageContext(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessageContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSplitKey(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		prefix string
		want   []string
	}{
		{
			name:   "valid key with prefix",
			key:    "bubbleId:chat1:bubble1",
			prefix: "bubbleId:",
			want:   []string{"", "chat1", "bubble1"},
		},
		{
			name:   "key without prefix",
			key:    "other:key",
			prefix: "bubbleId:",
			want:   nil,
		},
		{
			name:   "empty key",
			key:    "",
			prefix: "bubbleId:",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitKey(tt.key, tt.prefix)
			if len(got) != len(tt.want) {
				t.Errorf("splitKey() returned %d parts, want %d", len(got), len(tt.want))
				return
			}
			for i, w := range tt.want {
				if i < len(got) && got[i] != w {
					t.Errorf("splitKey() part[%d] = %q, want %q", i, got[i], w)
				}
			}
		})
	}
}

func TestParseRawBubble_InvalidKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{
			name:    "invalid key format",
			key:     "invalid:key",
			value:   `{"bubbleId":"bubble1"}`,
			wantErr: true,
		},
		{
			name:    "missing parts",
			key:     "bubbleId:chat1",
			value:   `{"bubbleId":"bubble1"}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			key:     "bubbleId:chat1:bubble1",
			value:   `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRawBubble(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRawBubble() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseRawComposer_InvalidKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{
			name:    "invalid key format",
			key:     "invalid:key",
			value:   `{"composerId":"composer1"}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			key:     "composerData:composer1",
			value:   `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRawComposer(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRawComposer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
