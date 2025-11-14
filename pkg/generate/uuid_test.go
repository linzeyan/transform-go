package generate

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var ulidPattern = regexp.MustCompile(`^[0-9A-Z]{26}$`)

func TestGenerateUUIDs(t *testing.T) {
	uuids, err := GenerateUUIDs()
	require.NoError(t, err)
	require.Len(t, uuids, 10)
	for version, val := range uuids {
		switch version {
		case "ulid":
			require.Equal(t, 26, len(val))
			require.True(t, ulidPattern.MatchString(val))
			continue
		}
		if version == "guid" {
			require.Equal(t, strings.ToUpper(val), val)
		}
		require.Equal(t, 36, len(val))
		require.True(t, strings.Contains(val, "-"))
		n := strings.Split(val, "-")
		require.Len(t, n, 5)
		switch version {
		case "v1":
			require.EqualValues(t, '1', val[14])
		case "v2":
			require.EqualValues(t, '2', val[14])
		case "v3":
			require.EqualValues(t, '3', val[14])
		case "v4":
			require.EqualValues(t, '4', val[14])
		case "v5":
			require.EqualValues(t, '5', val[14])
		case "v6":
			require.EqualValues(t, '6', val[14])
		case "v7":
			require.EqualValues(t, '7', val[14])
		case "v8":
			require.EqualValues(t, '8', val[14])
		case "guid":
			require.EqualValues(t, '4', strings.ToLower(val)[14])
		default:
			t.Fatalf("unexpected identifier %s", version)
		}
	}
}
