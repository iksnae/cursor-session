package internal

import (
	"fmt"
	"sort"
	"sync"
	"time"
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
	// NOTE: FullConversationHeadersOnly array is already in the correct chronological order.
	// We preserve this order and only sort by timestamp if timestamps differ.
	for _, header := range composer.FullConversationHeadersOnly {
		bubble, ok := r.bubbleMap.Get(header.BubbleID)
		if !ok {
			LogDebug("Bubble %s referenced in composer %s not found in bubble map", header.BubbleID, composer.ComposerID)
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

	// Sort messages by timestamp ONLY if timestamps differ
	// cursor-agent doesn't store per-message timestamps, so all messages have the same
	// session createdAt. In this case, we preserve the order from FullConversationHeadersOnly.
	hasDifferentTimestamps := false
	if len(conv.Messages) > 1 {
		firstTimestamp := conv.Messages[0].Timestamp
		for i := 1; i < len(conv.Messages); i++ {
			if conv.Messages[i].Timestamp != firstTimestamp {
				hasDifferentTimestamps = true
				break
			}
		}
	}

	if hasDifferentTimestamps {
		// Timestamps differ - sort by timestamp to ensure chronological order
		sort.Slice(conv.Messages, func(i, j int) bool {
			return conv.Messages[i].Timestamp < conv.Messages[j].Timestamp
		})
	}
	// If all timestamps are the same, preserve order from FullConversationHeadersOnly array
	// This is the correct order for cursor-agent sessions

	return conv, nil
}

// ReconstructAllConversations reconstructs all conversations from composers
func (r *Reconstructor) ReconstructAllConversations(composers []*RawComposer) ([]*ReconstructedConversation, error) {
	var conversations []*ReconstructedConversation

	for _, composer := range composers {
		conv, err := r.ReconstructConversation(composer)
		if err != nil {
			LogWarn("Failed to reconstruct conversation for composer %s: %v", composer.ComposerID, err)
			continue
		}

		// Only include conversations with messages
		if len(conv.Messages) == 0 {
			headerCount := len(composer.FullConversationHeadersOnly)
			LogWarn("Composer %s produced 0 messages (had %d headers). "+
				"Possible causes: headers reference non-existent bubbles, or all messages were empty",
				composer.ComposerID, headerCount)
			continue
		}
		conversations = append(conversations, conv)
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
	LogInfo("Built bubble map with %d bubbles", bubbleMap.Len())

	// Collect composers
	var composers []*RawComposer
	for composer := range composerChan {
		if composer != nil {
			composers = append(composers, composer)
			headerCount := len(composer.FullConversationHeadersOnly)
			LogInfo("Composer %s: %d headers, name='%s'", composer.ComposerID, headerCount, composer.Name)
		}
	}
	LogInfo("Collected %d composers from channel", len(composers))

	// Build context map from channel
	contextMap := make(map[string][]*MessageContext)
	contextCount := 0
	for context := range contextChan {
		if context != nil {
			contextMap[context.ComposerID] = append(contextMap[context.ComposerID], context)
			contextCount++
		}
	}
	LogInfo("Built context map with %d contexts across %d composers", contextCount, len(contextMap))

	// If no composers but we have bubbles, create composers from bubbles
	// This handles cursor-agent format where messages are stored as bubbles without explicit composers
	if len(composers) == 0 && bubbleMap.Len() > 0 {
		LogInfo("No composers found, creating composers from %d bubbles", bubbleMap.Len())
		composers = createComposersFromBubbles(bubbleMap)
		LogInfo("Created %d composer(s) from bubbles", len(composers))
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
			LogWarn("Failed to load bubbles: %v", err)
			return
		}
		LogInfo("Loading %d bubbles into channel", len(bubbles))
		for _, bubble := range bubbles {
			if bubble != nil {
				bubbleChan <- bubble
			}
		}
		LogInfo("Finished sending %d bubbles to channel", len(bubbles))
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

// createComposersFromBubbles creates composers from bubbles when no explicit composers exist
// This handles cursor-agent format where messages are stored as bubbles without composers
func createComposersFromBubbles(bubbleMap *BubbleMap) []*RawComposer {
	// Group bubbles by ChatID
	bubblesByChatID := make(map[string][]*RawBubble)
	allBubbles := bubbleMap.GetAll()
	for _, bubble := range allBubbles {
		chatID := bubble.ChatID
		if chatID == "" {
			// Use a default chatID if not set
			chatID = "default-session"
		}
		bubblesByChatID[chatID] = append(bubblesByChatID[chatID], bubble)
	}

	var composers []*RawComposer
	for chatID, bubbles := range bubblesByChatID {
		// NOTE: cursor-agent doesn't store per-message timestamps, so all bubbles have the same
		// session createdAt. We cannot sort by timestamp. Instead, we preserve the order from
		// the database query (which reflects insertion order). The database query should use
		// ORDER BY rowid to ensure consistent ordering, but even without it, SQLite typically
		// returns rows in insertion order.
		//
		// Only sort by timestamp if timestamps actually differ (shouldn't happen for cursor-agent)
		hasDifferentTimestamps := false
		if len(bubbles) > 1 {
			firstTimestamp := bubbles[0].Timestamp
			for i := 1; i < len(bubbles); i++ {
				if bubbles[i].Timestamp != firstTimestamp {
					hasDifferentTimestamps = true
					break
				}
			}
		}

		if hasDifferentTimestamps {
			// Timestamps differ - sort by timestamp
			sort.Slice(bubbles, func(i, j int) bool {
				return bubbles[i].Timestamp < bubbles[j].Timestamp
			})
		}
		// Otherwise, preserve the order from database (insertion order)

		// Create conversation headers from bubbles
		headers := make([]ConversationHeader, 0, len(bubbles))
		var createdAt, lastUpdatedAt int64
		for _, bubble := range bubbles {
			headers = append(headers, ConversationHeader{
				BubbleID: bubble.BubbleID,
				Type:     bubble.Type,
			})
			if createdAt == 0 || bubble.Timestamp < createdAt {
				createdAt = bubble.Timestamp
			}
			if bubble.Timestamp > lastUpdatedAt {
				lastUpdatedAt = bubble.Timestamp
			}
		}

		// If timestamps are 0, use current time
		if createdAt == 0 {
			createdAt = time.Now().UnixMilli()
		}
		if lastUpdatedAt == 0 {
			lastUpdatedAt = time.Now().UnixMilli()
		}

		composer := &RawComposer{
			ComposerID:                  chatID,
			Name:                        "", // Will be set from meta if available
			FullConversationHeadersOnly: headers,
			CreatedAt:                   createdAt,
			LastUpdatedAt:               lastUpdatedAt,
		}

		composers = append(composers, composer)
		LogInfo("Created composer %s from %d bubbles", chatID, len(headers))
	}

	return composers
}
