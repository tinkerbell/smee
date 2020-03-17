bindata.go: bin/ipxe.efi bin/snp-hua.efi bin/snp-nolacp.efi bin/undionly.kpxe
	go-bindata -pkg ipxe -o $@.tmp $^
	gofmt -w $@.tmp
	mv $@.tmp $@

deps := $(addprefix ipxe/, build.sh colour.h console.h general.aarch64-snp-nolacp.h general.efi.h general.hua.h general.undionly.h Makefile)

bin/snp-hua.efi bin/ipxe.efi bin/snp-nolacp.efi bin/undionly.kpxe: $(deps)

bin/ipxe.efi:
	$(MAKE) -C ipxe $(@F)
	cp ipxe/$(@F) $@

bin/snp-nolacp.efi:
	$(MAKE) -C ipxe $(@F)
	cp ipxe/$(@F) $@

bin/undionly.kpxe:
	$(MAKE) -C ipxe $(@F)
	cp ipxe/$(@F) $@

