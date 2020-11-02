let _pkgs = import <nixpkgs> { };
in { pkgs ? import (_pkgs.fetchFromGitHub {
  owner = "NixOS";
  repo = "nixpkgs";
  #branch@date: nixpkgs@2020-10-25
  rev = "1920b371c870f4e939e1e73ee83db9d8b6e0b217";
  sha256 = "0riyyxvygc7dbz2jh1n0gss1x03m0wh7dwqldczisx15hws3w3nm";
}) { } }:

with pkgs;

let

  mockgen = buildGoModule rec {
    pname = "mock";
    version = "1.4.3";

    doCheck = false;
    src = fetchFromGitHub {
      owner = "golang";
      repo = pname;
      rev = "v${version}";
      sha256 = "1p37xnja1dgq5ykx24n7wincwz2gahjh71b95p8vpw7ss2g8j8wx";
    };
    vendorSha256 = "1kpiij3pimwv3gn28rbrdvlw9q5c76lzw6zpa12q6pgck76acdw4";
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
