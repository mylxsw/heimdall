package commands_test

import (
	"testing"

	"github.com/mylxsw/go-utils/assert"
	"github.com/mylxsw/heimdall/commands"
)

func TestSplitNumRange(t *testing.T) {
	ranges := commands.SplitNumToRange(1005, 100)
	assert.Equal(t, 11, len(ranges))
	assert.Equal(t, 0, ranges[0].Start)
	assert.Equal(t, 100, ranges[1].Count)
	assert.Equal(t, 5, ranges[10].Count)
}
