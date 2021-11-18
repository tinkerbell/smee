package ipxe

import (
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestNonExistent(t *testing.T) {
	b, err := ReadFile("non-existent-filename-here-k-thx-bye")
	assert.Error(t, err)
	assert.Nil(t, b)
}

func TestFullPath(t *testing.T) {
	b, err := ReadFile("ipxe/snp-nolacp.efi")
	assert.NoError(t, err)
	assert.NotNil(t, b)
}
