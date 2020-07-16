let _pkgs = import <nixpkgs> { };
in { pkgs ? import (_pkgs.fetchFromGitHub {
  owner = "NixOS";
  repo = "nixpkgs-channels";
  #branch@date: nixpkgs-unstable@2020-02-01
  rev = "e3a9318b6fdb2b022c0bda66d399e1e481b24b5c";
  sha256 = "1hlblna9j0afvcm20p15f5is7cmwl96mc4vavc99ydc4yc9df62a";
}) { } }:

with pkgs;

let

  mockgen = buildGoModule rec {
    pname = "mock";
    version = "1.4.3";

    src = fetchFromGitHub {
      owner = "golang";
      repo = pname;
      rev = "v${version}";
      sha256 = "1p37xnja1dgq5ykx24n7wincwz2gahjh71b95p8vpw7ss2g8j8wx";
    };
    modSha256 = "0nfbh1sb4zh32xpfg25mbfnd4xflks95ram1m1rfdqbdwx2yc5jl";
    subPackages = [ "mockgen" ];
  };

in mkShell {
  buildInputs = [
    gcc
    git-lfs
    gnumake
    go
    go-bindata
    mockgen
    pkgsCross.aarch64-multiplatform.buildPackages.gcc
    protobuf
    xz
  ];
}
