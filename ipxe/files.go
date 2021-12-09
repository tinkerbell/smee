package ipxe

import "embed"

//go:embed ipxe/ipxe.efi
//go:embed ipxe/snp-hua.efi
//go:embed ipxe/snp-nolacp.efi
//go:embed ipxe/undionly.kpxe
var Files embed.FS
