package fs_metricsstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.skia.org/infra/go/firestore"
	"go.skia.org/infra/go/testutils/unittest"
	"go.skia.org/infra/golden/go/diff"
	"go.skia.org/infra/golden/go/types"
)

func TestToStoreEntryToDiffMetrics(t *testing.T) {
	unittest.SmallTest(t)
	expectedDiffMetrics := makeDiffMetrics(100)
	entry := toStoreEntry(expectedDiffMetrics)
	actualDiffMetrics := entry.toDiffMetrics()
	assert.Equal(t, expectedDiffMetrics, actualDiffMetrics)
}

func TestPutGetDiffMetrics(t *testing.T) {
	unittest.LargeTest(t)
	c, cleanup := firestore.NewClientForTesting(t)
	defer cleanup()

	// Create Firestore-backed MetricsStore instance.
	f := New(c)

	// Create test DiffMetrics instances.
	id1 := "abc-def"
	expectedDiffMetrics1 := makeDiffMetrics(100)
	id2 := "ghi-jkl"
	expectedDiffMetrics2 := makeDiffMetrics(200)

	// Not found.
	m, err := f.LoadDiffMetrics(id1)
	assert.NoError(t, err)
	assert.Nil(t, m)
	m, err = f.LoadDiffMetrics(id2)
	assert.NoError(t, err)
	assert.Nil(t, m)

	// Save them.
	err = f.SaveDiffMetrics(id1, expectedDiffMetrics1)
	assert.NoError(t, err)
	err = f.SaveDiffMetrics(id2, expectedDiffMetrics2)
	assert.NoError(t, err)

	// Load them.
	actualDiffMetrics1, err := f.LoadDiffMetrics(id1)
	assert.NoError(t, err)
	actualDiffMetrics2, err := f.LoadDiffMetrics(id2)
	assert.NoError(t, err)

	// Assert that the right diff metrics were returned.
	assert.Equal(t, expectedDiffMetrics1, actualDiffMetrics1)
	assert.Equal(t, expectedDiffMetrics2, actualDiffMetrics2)
}

func TestPurge(t *testing.T) {
	unittest.LargeTest(t)
	c, cleanup := firestore.NewClientForTesting(t)
	defer cleanup()

	// Create Firestore-backed MetricsStore instance.
	f := New(c)

	// Purge non-existent digest.
	err := f.PurgeDiffMetrics(types.DigestSlice{types.Digest("abc")})
	assert.NoError(t, err)

	// Add metrics.
	leftId := types.Digest("abc")
	rightId := types.Digest("def")
	diffId := string(leftId + "-" + rightId)
	expected := makeDiffMetrics(100)
	assert.NoError(t, f.SaveDiffMetrics(diffId, expected))

	// Purging by coercing the diffId as a types.Digest does nothing.
	err = f.PurgeDiffMetrics(types.DigestSlice{types.Digest(diffId)})
	assert.NoError(t, err)
	dm, err := f.LoadDiffMetrics(diffId)
	assert.NoError(t, err)
	assert.Equal(t, expected, dm)

	// Purging by leftId works.
	err = f.PurgeDiffMetrics(types.DigestSlice{leftId})
	assert.NoError(t, err)
	dm, err = f.LoadDiffMetrics(diffId)
	assert.NoError(t, err)
	assert.Nil(t, dm)

	// Re-add metric.
	assert.NoError(t, f.SaveDiffMetrics(diffId, expected))

	// Purging by rightId works.
	err = f.PurgeDiffMetrics(types.DigestSlice{rightId})
	assert.NoError(t, err)
	dm, err = f.LoadDiffMetrics(diffId)
	assert.NoError(t, err)
	assert.Nil(t, dm)
}

func TestPurgeMultiple(t *testing.T) {
	unittest.LargeTest(t)
	c, cleanup := firestore.NewClientForTesting(t)
	defer cleanup()

	// Create Firestore-backed MetricsStore instance.
	f := New(c)

	// Add multiple metrics.
	assert.NoError(t, f.SaveDiffMetrics("aaa-bbb", makeDiffMetrics(100)))
	assert.NoError(t, f.SaveDiffMetrics("aaa-ccc", makeDiffMetrics(200)))
	assert.NoError(t, f.SaveDiffMetrics("aaa-ddd", makeDiffMetrics(300)))
	assert.NoError(t, f.SaveDiffMetrics("aaa-eee", makeDiffMetrics(400)))
	assert.NoError(t, f.SaveDiffMetrics("bbb-ccc", makeDiffMetrics(500)))
	assert.NoError(t, f.SaveDiffMetrics("bbb-ddd", makeDiffMetrics(600)))
	assert.NoError(t, f.SaveDiffMetrics("bbb-eee", makeDiffMetrics(700)))
	assert.NoError(t, f.SaveDiffMetrics("ccc-ddd", makeDiffMetrics(800)))
	assert.NoError(t, f.SaveDiffMetrics("ccc-eee", makeDiffMetrics(900)))
	assert.NoError(t, f.SaveDiffMetrics("ddd-eee", makeDiffMetrics(1000)))

	// Purge some but not all.
	err := f.PurgeDiffMetrics(types.DigestSlice{
		types.Digest("aaa"),
		types.Digest("bbb"),
	})
	assert.NoError(t, err)

	// Assert that the expected metrics were purged.
	purged := []string{
		"aaa-bbb",
		"aaa-ccc",
		"aaa-ddd",
		"aaa-eee",
		"bbb-ccc",
		"bbb-ddd",
		"bbb-eee",
	}
	for _, id := range purged {
		dm, err := f.LoadDiffMetrics(id)
		assert.NoError(t, err)
		assert.Nil(t, dm)
	}

	// Assert that the expected metrics remain in the store.
	notPurged := map[string]*diff.DiffMetrics{
		"ccc-ddd": makeDiffMetrics(800),
		"ccc-eee": makeDiffMetrics(900),
		"ddd-eee": makeDiffMetrics(1000),
	}
	for id, expectedDiffMetrics := range notPurged {
		actualDiffMetrics, err := f.LoadDiffMetrics(id)
		assert.NoError(t, err)
		assert.Equal(t, expectedDiffMetrics, actualDiffMetrics)
	}
}

func makeDiffMetrics(numDiffPixels int) *diff.DiffMetrics {
	diffMetrics := &diff.DiffMetrics{
		NumDiffPixels:    numDiffPixels,
		PixelDiffPercent: 0.5,
		MaxRGBADiffs:     []int{2, 3, 5, 7},
		DimDiffer:        true,
		Diffs: map[string]float32{
			diff.METRIC_PERCENT: 0.5,
			diff.METRIC_PIXEL:   float32(numDiffPixels),
		},
	}
	diffMetrics.Diffs[diff.METRIC_COMBINED] = diff.CombinedDiffMetric(diffMetrics, nil, nil)
	return diffMetrics
}