bindata.go: bin/snp-hua.efi bin/ipxe.efi bin/snp-nolacp.efi bin/undionly.kpxe
	# TODO use this commit time of this commit to set as the mod time
	# fail on dirty?
	go-bindata -pkg ipxe bin/*
	gofmt -w .

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

