package nixos

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var oshwToInitPath = map[string]string{
	"nixos_17_03/c1.small.x86":  "/nix/store/a8nhjab9brxw80lnvrpxj37wkgmxa0bl-nixos-system-ipxe-17.03.945.5acb454e2a/init",
	"nixos_17_03/c1.xlarge.x86": "/nix/store/a8nhjab9brxw80lnvrpxj37wkgmxa0bl-nixos-system-ipxe-17.03.945.5acb454e2a/init",
	"nixos_17_03/m1.xlarge.x86": "/nix/store/a8nhjab9brxw80lnvrpxj37wkgmxa0bl-nixos-system-ipxe-17.03.945.5acb454e2a/init",
	"nixos_17_03/t1.small.x86":  "/nix/store/a8nhjab9brxw80lnvrpxj37wkgmxa0bl-nixos-system-ipxe-17.03.945.5acb454e2a/init",

	"nixos_18_03/c1.large.arm":  "/nix/store/gbhizlyjj5gc3fayvw79dsii6ac5yb74-nixos-system-install-environment-18.03.132610.49a6964a425/init",
	"nixos_18_03/c1.small.x86":  "/nix/store/hq6hni37qjql1206j7hkhqf1x017w8qz-nixos-system-install-environment-18.03.132610.49a6964a425/init",
	"nixos_18_03/c1.xlarge.x86": "/nix/store/x8lvlh5c5rdfaf25w50fpdylkcwd3ihy-nixos-system-install-environment-18.03.132610.49a6964a425/init",
	"nixos_18_03/c2.medium.x86": "/nix/store/46mmhc2jv2wkkda90dqks1p2054irszy-nixos-system-install-environment-18.03.132610.49a6964a425/init",
	"nixos_18_03/m1.xlarge.x86": "/nix/store/59v36skcl0ymsq61phx5yxifn89ddi9n-nixos-system-install-environment-18.03.132610.49a6964a425/init",
	"nixos_18_03/m2.xlarge.x86": "/nix/store/zizskvd3hb9arcn7lswqy1j81p538q1w-nixos-system-install-environment-18.03.132610.49a6964a425/init",
	"nixos_18_03/s1.large.x86":  "/nix/store/nnic31dppmzamq7l3sn3iyjks55qsrn5-nixos-system-install-environment-18.03.132610.49a6964a425/init",
	"nixos_18_03/t1.small.x86":  "/nix/store/9zpihimwsjysscvidjs1dfa0zwfnxim0-nixos-system-install-environment-18.03.132610.49a6964a425/init",
	"nixos_18_03/x1.small.x86":  "/nix/store/bd42lgd9rmz4xmq3zgs8j31rf0g7fn4q-nixos-system-install-environment-18.03.132610.49a6964a425/init",

	"nixos_19_03/c1.large.arm":  "/nix/store/jl25bvf4c72b03scsw46g8m961zy7gyc-nixos-system-install-environment-19.03pre-git/init",
	"nixos_19_03/c1.small.x86":  "/nix/store/lyksilpbwywk5p33m9yycqgib0v62c6i-nixos-system-install-environment-19.03pre-git/init",
	"nixos_19_03/c1.xlarge.x86": "/nix/store/5vx4mmvfdcsxxbklw5n9mpkbq7i26h9w-nixos-system-install-environment-19.03pre-git/init",
	"nixos_19_03/c2.large.arm":  "/nix/store/a2qi1jc9dfag50ql4nl037zcgcgqlli0-nixos-system-install-environment-19.03pre-git/init",
	"nixos_19_03/c2.medium.x86": "/nix/store/rmc6ykrxyx01r0iknym4h5gjhk98w7vx-nixos-system-install-environment-19.03pre-git/init",
	"nixos_19_03/g2.large.x86":  "/nix/store/pdsphmz3v6nj219pn0i9pxlilm4y0y9j-nixos-system-install-environment-19.03pre-git/init",
	"nixos_19_03/m1.xlarge.x86": "/nix/store/0rq48c0d3nzzcndz3vmg5a00xyz9cj96-nixos-system-install-environment-19.03pre-git/init",
	"nixos_19_03/m2.xlarge.x86": "/nix/store/ygkc1kh4ckfkf3qav0s4psf4dz605p6z-nixos-system-install-environment-19.03pre-git/init",
	"nixos_19_03/n2.xlarge.x86": "/nix/store/ygkc1kh4ckfkf3qav0s4psf4dz605p6z-nixos-system-install-environment-19.03pre-git/init",
	"nixos_19_03/s1.large.x86":  "/nix/store/cy3vmnkyvn6zwl21r0pafjkql8332hmj-nixos-system-install-environment-19.03pre-git/init",
	"nixos_19_03/t1.small.x86":  "/nix/store/wim9i57y34n133js866x7xldpgc6za2k-nixos-system-install-environment-19.03pre-git/init",
	"nixos_19_03/x1.small.x86":  "/nix/store/zm24a2hgwdd9wgbr012qwh53lf2g62nv-nixos-system-install-environment-19.03pre-git/init",
	"nixos_19_03/x2.xlarge.x86": "/nix/store/hnc28768rvnh6m3cwn6h0xg6frxlnws8-nixos-system-install-environment-19.03pre-git/init",
}

func TestBuildPaths(t *testing.T) {
	assert := require.New(t)

	assert.NoError(os.Setenv("nixos_17_03__c1_small_x86", "a8nhjab9brxw80lnvrpxj37wkgmxa0bl-nixos-system-ipxe-17.03.945.5acb454e2a"))
	assert.NoError(os.Setenv("nixos_17_03__c1_xlarge_x86", "a8nhjab9brxw80lnvrpxj37wkgmxa0bl-nixos-system-ipxe-17.03.945.5acb454e2a"))
	assert.NoError(os.Setenv("nixos_17_03__m1_xlarge_x86", "a8nhjab9brxw80lnvrpxj37wkgmxa0bl-nixos-system-ipxe-17.03.945.5acb454e2a"))
	assert.NoError(os.Setenv("nixos_17_03__t1_small_x86", "a8nhjab9brxw80lnvrpxj37wkgmxa0bl-nixos-system-ipxe-17.03.945.5acb454e2a"))

	assert.NoError(os.Setenv("nixos_18_03__c1_large_arm", "gbhizlyjj5gc3fayvw79dsii6ac5yb74-nixos-system-install-environment-18.03.132610.49a6964a425"))
	assert.NoError(os.Setenv("nixos_18_03__c1_small_x86", "hq6hni37qjql1206j7hkhqf1x017w8qz-nixos-system-install-environment-18.03.132610.49a6964a425"))
	assert.NoError(os.Setenv("nixos_18_03__c1_xlarge_x86", "x8lvlh5c5rdfaf25w50fpdylkcwd3ihy-nixos-system-install-environment-18.03.132610.49a6964a425"))
	assert.NoError(os.Setenv("nixos_18_03__c2_medium_x86", "46mmhc2jv2wkkda90dqks1p2054irszy-nixos-system-install-environment-18.03.132610.49a6964a425"))
	assert.NoError(os.Setenv("nixos_18_03__m1_xlarge_x86", "59v36skcl0ymsq61phx5yxifn89ddi9n-nixos-system-install-environment-18.03.132610.49a6964a425"))
	assert.NoError(os.Setenv("nixos_18_03__m2_xlarge_x86", "zizskvd3hb9arcn7lswqy1j81p538q1w-nixos-system-install-environment-18.03.132610.49a6964a425"))
	assert.NoError(os.Setenv("nixos_18_03__s1_large_x86", "nnic31dppmzamq7l3sn3iyjks55qsrn5-nixos-system-install-environment-18.03.132610.49a6964a425"))
	assert.NoError(os.Setenv("nixos_18_03__t1_small_x86", "9zpihimwsjysscvidjs1dfa0zwfnxim0-nixos-system-install-environment-18.03.132610.49a6964a425"))
	assert.NoError(os.Setenv("nixos_18_03__x1_small_x86", "bd42lgd9rmz4xmq3zgs8j31rf0g7fn4q-nixos-system-install-environment-18.03.132610.49a6964a425"))

	assert.NoError(os.Setenv("nixos_19_03__c1_large_arm", "jl25bvf4c72b03scsw46g8m961zy7gyc-nixos-system-install-environment-19.03pre-git"))
	assert.NoError(os.Setenv("nixos_19_03__c1_small_x86", "lyksilpbwywk5p33m9yycqgib0v62c6i-nixos-system-install-environment-19.03pre-git"))
	assert.NoError(os.Setenv("nixos_19_03__c1_xlarge_x86", "5vx4mmvfdcsxxbklw5n9mpkbq7i26h9w-nixos-system-install-environment-19.03pre-git"))
	assert.NoError(os.Setenv("nixos_19_03__c2_large_arm", "a2qi1jc9dfag50ql4nl037zcgcgqlli0-nixos-system-install-environment-19.03pre-git"))
	assert.NoError(os.Setenv("nixos_19_03__c2_medium_x86", "rmc6ykrxyx01r0iknym4h5gjhk98w7vx-nixos-system-install-environment-19.03pre-git"))
	assert.NoError(os.Setenv("nixos_19_03__g2_large_x86", "pdsphmz3v6nj219pn0i9pxlilm4y0y9j-nixos-system-install-environment-19.03pre-git"))
	assert.NoError(os.Setenv("nixos_19_03__m1_xlarge_x86", "0rq48c0d3nzzcndz3vmg5a00xyz9cj96-nixos-system-install-environment-19.03pre-git"))
	assert.NoError(os.Setenv("nixos_19_03__m2_xlarge_x86", "ygkc1kh4ckfkf3qav0s4psf4dz605p6z-nixos-system-install-environment-19.03pre-git"))
	assert.NoError(os.Setenv("nixos_19_03__n2_xlarge_x86", "ygkc1kh4ckfkf3qav0s4psf4dz605p6z-nixos-system-install-environment-19.03pre-git"))
	assert.NoError(os.Setenv("nixos_19_03__s1_large_x86", "cy3vmnkyvn6zwl21r0pafjkql8332hmj-nixos-system-install-environment-19.03pre-git"))
	assert.NoError(os.Setenv("nixos_19_03__t1_small_x86", "wim9i57y34n133js866x7xldpgc6za2k-nixos-system-install-environment-19.03pre-git"))
	assert.NoError(os.Setenv("nixos_19_03__x1_small_x86", "zm24a2hgwdd9wgbr012qwh53lf2g62nv-nixos-system-install-environment-19.03pre-git"))
	assert.NoError(os.Setenv("nixos_19_03__x2_xlarge_x86", "hnc28768rvnh6m3cwn6h0xg6frxlnws8-nixos-system-install-environment-19.03pre-git"))

	assert.Equal(oshwToInitPath, BuildInitPaths())
}
