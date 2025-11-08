package internal

import (
	"fmt"
	"sort"
	"sync"
)

// ReconstructedConversation represents a fully reconstructed conversation
type ReconstructedConversation struct {
	ComposerID string
	Name       string
	Messages   []ReconstructedMessage
	CreatedAt  int64
	UpdatedAt  int64
}

// ReconstructedMessage represents a message in a reconstructed conversation
type ReconstructedMessage struct {
	BubbleID  string
	Type      int // 1=user, 2=assistant
	Text      string
	Timestamp int64
	Context   *MessageContext
}

// Reconstructor handles conversation reconstruction
type Reconstructor struct {
	bubbleMap  *BubbleMap
	contextMap map[string][]*MessageContext
}

// NewReconstructor creates a new Reconstructor
func NewReconstructor(bubbleMap *BubbleMap, contextMap map[string][]*MessageContext) *Reconstructor {
	return &Reconstructor{
		bubbleMap:  bubbleMap,
		contextMap: contextMap,
	}
}

// ReconstructConversation reconstructs a conversation from a composer
func (r *Reconstructor) ReconstructConversation(composer *RawComposer) (*ReconstructedConversation, error) {
	if composer == nil {
		return nil, fmt.Errorf("composer is nil")
	}

	conv := &ReconstructedConversation{
		ComposerID: composer.ComposerID,
		Name:       composer.Name,
		CreatedAt:  composer.CreatedAt,
		UpdatedAt:  composer.LastUpdatedAt,
	}

	// Get context for this composer
	contexts := r.contextMap[composer.ComposerID]
	contextByBubbleID := make(map[string]*MessageContext)
	for _, ctx := range contexts {
		contextByBubbleID[ctx.BubbleID] = ctx
	}

	// Reconstruct messages from headers
	for _, header := range composer.FullConversationHeadersOnly {
		bubble, ok := r.bubbleMap.Get(header.BubbleID)
		if !ok {
			// Bubble not found, skip
			continue
		}

		// Extract text from bubble
		text, err := ExtractTextFromBubble(bubble)
		if err != nil {
			// Log error but continue
			LogDebug("Failed to extract text from bubble %s: %v", header.BubbleID, err)
			continue
		}

		// Skip empty messages (matching reference implementation behavior)
		// Only skip if it's the placeholder, not if it's actual empty content
		if text == "" || text == "[Message with no extractable text content]" {
			LogDebug("Skipping empty message bubble %s", header.BubbleID)
			continue
		}

		// Get context for this bubble
		context := contextByBubbleID[header.BubbleID]

		msg := ReconstructedMessage{
			BubbleID:  header.BubbleID,
			Type:      header.Type,
			Text:      text,
			Timestamp: bubble.Timestamp,
			Context:   context,
		}

		conv.Messages = append(conv.Messages, msg)
	}

	// Sort messages by timestamp
	sort.Slice(conv.Messages, func(i, j int) bool {
		return conv.Messages[i].Timestamp < conv.Messages[j].Timestamp
	})

	return conv, nil
}

// ReconstructAllConversations reconstructs all conversations from composers
func (r *Reconstructor) ReconstructAllConversations(composers []*RawComposer) ([]*ReconstructedConversation, error) {
	var conversations []*ReconstructedConversation

	for _, composer := range composers {
		conv, err := r.ReconstructConversation(composer)
		if err != nil {
			// Log error but continue
			continue
		}

		// Only include conversations with messages
		if len(conv.Messages) > 0 {
			conversations = append(conversations, conv)
		}
	}

	return conversations, nil
}

// ReconstructAsync reconstructs conversations using async processing
func ReconstructAsync(
	bubbleChan <-chan *RawBubble,
	composerChan <-chan *RawComposer,
	contextChan <-chan *MessageContext,
) ([]*ReconstructedConversation, error) {
	// Build bubble map from channel
	bubbleMap := BuildBubbleMapFromChannel(bubbleChan)

	// Collect composers
	var composers []*RawComposer
	for composer := range composerChan {
		if composer != nil {
			composers = append(composers, composer)
		}
	}

	// Build context map from channel
	contextMap := make(map[string][]*MessageContext)
	for context := range contextChan {
		if context != nil {
			contextMap[context.ComposerID] = append(contextMap[context.ComposerID], context)
		}
	}

	// Reconstruct conversations
	reconstructor := NewReconstructor(bubbleMap, contextMap)
	return reconstructor.ReconstructAllConversations(composers)
}

// LoadDataAsync loads all data asynchronously and sends to channels
func LoadDataAsync(storage *Storage) (<-chan *RawBubble, <-chan *RawComposer, <-chan *MessageContext, error) {
	return LoadDataAsyncFromBackend(storage)
}

// LoadDataAsyncFromBackend loads all data asynchronously from a StorageBackend and sends to channels
func LoadDataAsyncFromBackend(backend StorageBackend) (<-chan *RawBubble, <-chan *RawComposer, <-chan *MessageContext, error) {
	bubbleChan := make(chan *RawBubble, 100)
	composerChan := make(chan *RawComposer, 100)
	contextChan := make(chan *MessageContext, 100)

	var wg sync.WaitGroup

	// Load bubbles
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(bubbleChan)
		bubbles, err := backend.LoadBubbles()
		if err != nil {
			return
		}
		for _, bubble := range bubbles {
			bubbleChan <- bubble
		}
	}()

	// Load composers
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(composerChan)
		composers, err := backend.LoadComposers()
		if err != nil {
			return
		}
		for _, composer := range composers {
			composerChan <- composer
		}
	}()

	// Load contexts
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(contextChan)
		contexts, err := backend.LoadMessageContexts()
		if err != nil {
			return
		}
		for _, contextList := range contexts {
			for _, ctx := range contextList {
				contextChan <- ctx
			}
		}
	}()

	// Wait for all to complete
	go func() {
		wg.Wait()
	}()

	return bubbleChan, composerChan, contextChan, nil
}
