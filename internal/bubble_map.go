package internal

import (
	"sync"
)

// BubbleMap provides thread-safe access to bubbles
type BubbleMap struct {
	mu      sync.RWMutex
	bubbles map[string]*RawBubble
}

// NewBubbleMap creates a new BubbleMap
func NewBubbleMap() *BubbleMap {
	return &BubbleMap{
		bubbles: make(map[string]*RawBubble),
	}
}

// Get retrieves a bubble by ID
func (bm *BubbleMap) Get(bubbleID string) (*RawBubble, bool) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	bubble, ok := bm.bubbles[bubbleID]
	return bubble, ok
}

// Set stores a bubble
func (bm *BubbleMap) Set(bubbleID string, bubble *RawBubble) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.bubbles[bubbleID] = bubble
}

// Len returns the number of bubbles in the map
func (bm *BubbleMap) Len() int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return len(bm.bubbles)
}

// GetAll returns all bubbles in the map
func (bm *BubbleMap) GetAll() []*RawBubble {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	bubbles := make([]*RawBubble, 0, len(bm.bubbles))
	for _, bubble := range bm.bubbles {
		bubbles = append(bubbles, bubble)
	}
	return bubbles
}

// BuildBubbleMapFromChannel builds a bubble map from a channel
func BuildBubbleMapFromChannel(bubbleChan <-chan *RawBubble) *BubbleMap {
	bm := NewBubbleMap()
	for bubble := range bubbleChan {
		if bubble != nil {
			bm.Set(bubble.BubbleID, bubble)
		}
	}
	return bm
}
