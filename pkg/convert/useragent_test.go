package convert

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateUserAgents(t *testing.T) {
	list, err := GenerateUserAgents("Chrome", "Windows")
	require.NoError(t, err)
	require.NotEmpty(t, list)
	for _, entry := range list {
		require.Contains(t, entry.BrowserName, "Chrome")
		require.NotEmpty(t, entry.UserAgent)
	}
}
