package machine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.skia.org/infra/go/deepequal/assertdeep"
	"go.skia.org/infra/go/testutils/unittest"
)

var testTime = time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)

func TestCopy(t *testing.T) {
	unittest.SmallTest(t)
	in := Description{
		Mode: ModeAvailable,
		Annotation: Annotation{
			Message:   "take offline",
			User:      "barney@example.com",
			Timestamp: testTime,
		},
		Dimensions: SwarmingDimensions{
			"foo": []string{"bar"},
		},
		LastUpdated: testTime,
	}
	out := in.Copy()
	require.Equal(t, in, out)
	assertdeep.Copy(t, in, out)

	// Confirm that the two Dimensions are separate.
	in.Dimensions["baz"] = []string{"quux"}
	require.NotEqual(t, in, out)
}