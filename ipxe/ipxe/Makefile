# ipxe commit and sha512sum
v := 18dc73d27edb55ebe9cb13c58d59af3da3bd374b
h := 17b6bbad8f0a94b15cdb5470bb60c8c7868759efd89f0ea8111c94eefaaa4c0c665a5b8d9547defeb8caf432a0d9d12e25aa81709b4e5a6055cb416c140b4de7

configs := colour.h console.h general.aarch64-snp-nolacp.h general.efi.h general.undionly.h

all: undionly.kpxe ipxe.efi snp-nolacp.efi

ipxe.efi: ipxe-x86_64-efi/src/bin-x86_64-efi/ipxe.efi
snp-nolacp.efi: ipxe-arm64-efi/src/bin-arm64-efi/snp.efi
undionly.kpxe: ipxe-x86_64-kpxe/src/bin/undionly.kpxe
ipxe.efi snp-nolacp.efi undionly.kpxe:
	cp $^ $@

ipxe-arm64-efi/src/bin-arm64-efi/snp.efi: ipxe-arm64-efi
ipxe-x86_64-efi/src/bin-x86_64-efi/ipxe.efi: ipxe-x86_64-efi
ipxe-x86_64-kpxe/src/bin/undionly.kpxe: ipxe-x86_64-kpxe

ipxe-arm64-efi/src/bin-arm64-efi/snp.efi ipxe-x86_64-efi/src/bin-x86_64-efi/ipxe.efi ipxe-x86_64-kpxe/src/bin/undionly.kpxe: build.sh ${configs}
	cp ${configs} $(dir $(patsubst %/,%,$(dir $@)))/config/local
	+./build.sh $@ $v

ipxe-arm64-efi ipxe-x86_64-efi ipxe-x86_64-kpxe: $v.tar.gz
	mkdir -p tmp-$@
	tar -zxf $v.tar.gz -C tmp-$@
	mv tmp-$@/ipxe-$v $@
	rm -rf tmp-$@

$v.tar.gz:
	curl -fL https://github.com/ipxe/ipxe/archive/$v.tar.gz > $v
	echo "$h  $v" | sha512sum -c
	mv $v $@
