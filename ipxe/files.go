package ipxe

import (
	"embed"
	"io/fs"
	"path"

	"github.com/pkg/errors"
)

//go:embed ipxe/ipxe.efi
//go:embed ipxe/snp-hua.efi
//go:embed ipxe/snp-nolacp.efi
//go:embed ipxe/undionly.kpxe
var files embed.FS

// ReadFile reads and returns the content of the named file.
// The named file must be just the basename of the intended file.
func ReadFile(name string) ([]byte, error) {
	name = path.Join("ipxe", name)
	b, err := files.ReadFile(name)
	if err != nil {
		return nil, errors.Wrap(err, "opening file")
	}

	return b, err
}

// Files returns the embedded files as an fs.FS re-rooted under the ipxe subtree.
func Files() fs.FS {
	sub, err := fs.Sub(files, "ipxe")
	if err != nil {
		panic(errors.Wrap(err, "re-rooting files"))
	}

	return sub
}
