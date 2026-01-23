package utils

import (
	"sync"
	"time"
)

var (
	processedMessages = make(map[string]time.Time)
	mu                sync.RWMutex
)

// IsDuplicate checks if a message ID has been processed recently (within 5 minutes)
// Returns true if the message is a duplicate and should be ignored
func IsDuplicate(msgID string) bool {
	if msgID == "" {
		return false
	}

	mu.RLock()
	timestamp, exists := processedMessages[msgID]
	mu.RUnlock()

	if exists && time.Since(timestamp) < 5*time.Minute {
		return true
	}

	mu.Lock()
	processedMessages[msgID] = time.Now()

	// Cleanup old entries if map gets too big
	if len(processedMessages) > 10000 {
		for k, v := range processedMessages {
			if time.Since(v) > 10*time.Minute {
				delete(processedMessages, k)
			}
		}
	}
	mu.Unlock()

	return false
}
