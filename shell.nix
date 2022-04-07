let _pkgs = import <nixpkgs> { };
in { pkgs ? import (_pkgs.fetchFromGitHub {
  owner = "NixOS";
  repo = "nixpkgs";
  #branch@date: nixpkgs@2020-11-24
  rev = "6625284c397b44bc9518a5a1567c1b5aae455c08";
  sha256 = "1w0czzv53sg35gp7sr506facbmzd33jm34p6cg23fb9kz5rf5c89";
}) { } }:

with pkgs;

mkShell {
  buildInputs = [
    curl
    expect
    gcc
    git
    git-lfs
    gnumake
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
