package unit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yugasun/hubsync/pkg/sync"
)

// TestOutputItemStruct tests the OutputItem struct functionality
func TestOutputItemStruct(t *testing.T) {
	// Recreate the original test case
	item := sync.OutputItem{Source: "src", Target: "tgt", Repository: "repo"}
	assert.Equal(t, "src", item.Source)
	assert.Equal(t, "tgt", item.Target)
	assert.Equal(t, "repo", item.Repository)

	// Add additional tests for the enhanced structure
	now := time.Now()
	end := now.Add(5 * time.Second)
	itemWithTime := sync.OutputItem{
		Source:     "src",
		Target:     "tgt",
		Repository: "repo",
		StartTime:  now,
		EndTime:    end,
		Duration:   end.Sub(now),
	}

	assert.Equal(t, "src", itemWithTime.Source)
	assert.Equal(t, "tgt", itemWithTime.Target)
	assert.Equal(t, "repo", itemWithTime.Repository)
	assert.Equal(t, now, itemWithTime.StartTime)
	assert.Equal(t, end, itemWithTime.EndTime)
	assert.Equal(t, 5*time.Second, itemWithTime.Duration)
}
