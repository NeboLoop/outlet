package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The nine outreach tools register, list over MCP with their input schemas,
// and carry the read-only and destructive hints clients use to ask first.
func TestOutboundToolsList(t *testing.T) {
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "outlet", Version: "test"}, nil)
	RegisterOutboundTools(server, nil)
	ct, st := mcp.NewInMemoryTransports()
	_, err := server.Connect(ctx, st, nil)
	require.NoError(t, err)
	client := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "test"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { cs.Close() })

	res, err := cs.ListTools(ctx, nil)
	require.NoError(t, err)
	byName := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		byName[tool.Name] = tool
	}
	require.Len(t, byName, len(outboundActions))
	for name := range outboundActions {
		require.Contains(t, byName, name)
		assert.NotNil(t, byName[name].InputSchema, name)
	}
	assert.True(t, byName["outreach_overview"].Annotations.ReadOnlyHint)
	assert.True(t, *byName["review_drafts"].Annotations.DestructiveHint)
	assert.False(t, *byName["draft_outreach"].Annotations.DestructiveHint)
}
