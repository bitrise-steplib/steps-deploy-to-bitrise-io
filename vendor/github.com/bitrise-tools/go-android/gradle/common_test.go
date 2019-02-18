package gradle

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCleanStringSlice(t *testing.T) {
	require.Equal(t, cleanStringSlice([]string{"", ""}), []string(nil))
	require.Equal(t, cleanStringSlice([]string{"", "test"}), []string{"test"})
	require.Equal(t, cleanStringSlice([]string{"", "   space "}), []string{"space"})
	require.Equal(t, cleanStringSlice([]string{"  ", "   space ", "   space2 "}), []string{"space", "space2"})
}
