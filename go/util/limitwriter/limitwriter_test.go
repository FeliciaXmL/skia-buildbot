package limitwriter

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.skia.org/infra/go/testutils/unittest"
)

func TestLimitBuf(t *testing.T) {
	unittest.SmallTest(t)
	buf := &bytes.Buffer{}
	b := New(buf, 10)
	n, err := b.Write([]byte("123456"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, 6, buf.Len())

	n, err = b.Write([]byte("123456"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, 10, buf.Len())
	assert.Equal(t, "1234561234", buf.String())

	n, err = b.Write([]byte("123456"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, 10, buf.Len())
	assert.Equal(t, "1234561234", buf.String())
}
