package ipxe

import (
	"embed"
	"io/fs"

	"github.com/pkg/errors"
)

//go:embed ipxe/ipxe.efi
//go:embed ipxe/snp-hua.efi
//go:embed ipxe/snp-nolacp.efi
//go:embed ipxe/undionly.kpxe
var files embed.FS

// ReadFile reads and returns the content of the named file.
func ReadFile(name string) ([]byte, error) {
	b, err := files.ReadFile(name)
	if err != nil {
		return nil, errors.Wrap(err, "opening file")
	}

	return b, err
}

// Files returns the embedded files as an fs.FS.
func Files() fs.FS {
	return files
}
