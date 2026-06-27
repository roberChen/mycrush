package customtools

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterAndLookup(t *testing.T) {
	t.Parallel()

	t.Cleanup(ClearRegisteredToolNames)

	require.False(t, IsRegisteredToolName("foo"))

	RegisterToolName("foo")
	RegisterToolName("bar")

	require.True(t, IsRegisteredToolName("foo"))
	require.True(t, IsRegisteredToolName("bar"))
	require.False(t, IsRegisteredToolName("baz"))

	ClearRegisteredToolNames()
	require.False(t, IsRegisteredToolName("foo"))
	require.False(t, IsRegisteredToolName("bar"))
}
