package uploaders

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_reconstructMultipartETag(t *testing.T) {
	t.Run("combines part ETags ordered by part number", func(t *testing.T) {
		// Parts out of order, with quoted hex MD5 ETags, must still produce the
		// part-number-ordered result.
		given := []UploadedPart{
			{PartNumber: 2, ETag: `"7ce8636c076f5f42316676f7ca5ccfbe"`}, // md5("lo")
			{PartNumber: 1, ETag: `"46356afe55fa3cea9cbe73ad442cad47"`}, // md5("hel")
		}

		got, err := reconstructMultipartETag(given)

		require.NoError(t, err)
		require.Equal(t, "554a2f6105cc700b8cc987b5ddfb8102-2", got)
	})

	t.Run("non-hex part ETag returns an error", func(t *testing.T) {
		given := []UploadedPart{{PartNumber: 1, ETag: "not-a-hex-digest"}}

		_, err := reconstructMultipartETag(given)

		require.Error(t, err)
	})
}
