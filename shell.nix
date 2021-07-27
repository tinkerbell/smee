let _pkgs = import <nixpkgs> { };
in { pkgs ? import (_pkgs.fetchFromGitHub {
  owner = "NixOS";
  repo = "nixpkgs";
  #branch@date: nixpkgs@2020-11-24
  rev = "6625284c397b44bc9518a5a1567c1b5aae455c08";
  sha256 = "1w0czzv53sg35gp7sr506facbmzd33jm34p6cg23fb9kz5rf5c89";
}) { } }:

with pkgs;

let
  custom_pkgs = import (_pkgs.fetchFromGitHub {
    # go 1.16.3
    owner = "NixOS";
    repo = "nixpkgs";
    #branch@date: nixpkgs-unstable@2021-04-19
    rev = "c92ca95afb5043bc6faa0d526460584eccff2277";
    sha256 = "14vmijjjypd4b3fcvxzi53n7i5g3l5x9ih0ci1j6h1m9j5fkh9iv";
  }) { };

  go_1_16_3 = custom_pkgs.go;

in mkShell {
  buildInputs = [
    curl
    expect
    gcc
    git
    git-lfs
    gnumake
    go_1_16_3
    golangci-lint
    nixfmt
    nodePackages.prettier
    perl
    protobuf
    qemu
    shellcheck
    shfmt
    xz
  ] ++ lib.optionals stdenv.isLinux
    [ pkgsCross.aarch64-multiplatform.buildPackages.gcc ];
}

