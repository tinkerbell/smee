let _pkgs = import <nixpkgs> { };
in { pkgs ? import (_pkgs.fetchFromGitHub {
  owner = "NixOS";
  repo = "nixpkgs";
  #branch@date: nixpkgs@2020-10-25
  rev = "1920b371c870f4e939e1e73ee83db9d8b6e0b217";
  sha256 = "0riyyxvygc7dbz2jh1n0gss1x03m0wh7dwqldczisx15hws3w3nm";
}) { } }:

with pkgs;

mkShell {
  buildInputs = [
    gcc
    git-lfs
    gnumake
    go
    golangci-lint
    go-bindata
    mockgen
    nodePackages.prettier
    pkgsCross.aarch64-multiplatform.buildPackages.gcc
    protobuf
    shfmt
    shellcheck
    xz
  ];
}
