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

