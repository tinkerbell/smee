let _pkgs = import <nixpkgs> { };
in { pkgs ? import (_pkgs.fetchFromGitHub {
  owner = "NixOS";
  repo = "nixpkgs";
  #branch@date: nixos-unstable-small@2022-04-18
  rev = "e33fe968df5a2503290682278399b1198f7ba56f";
  sha256 = "0kr30yj9825jx4zzcyn43c398mx3l63ndgfrg1y9v3d739mfgyw3";
}) { } }:

with pkgs;

mkShell {
  buildInputs = [
    git
    gnumake
    go_1_18
    nixfmt
    nodePackages.prettier
    perl
    protobuf
    shellcheck
    shfmt
  ];
}
