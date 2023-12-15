let _pkgs = import <nixpkgs> { };
in { pkgs ? import (_pkgs.fetchFromGitHub {
  owner = "NixOS";
  repo = "nixpkgs";
  #branch@date: nixpkgs-unstable@2023-10-08T09:04:33+02:00
  rev = "9957cd48326fe8dbd52fdc50dd2502307f188b0d";
  sha256 = "1l2hq1n1jl2l64fdcpq3jrfphaz10sd1cpsax3xdya0xgsncgcsi";
}) { } }:

with pkgs;

mkShell {
  buildInputs = [
    git
    gnumake
    go_1_21
    nixfmt
    nodePackages.prettier
    perl
    protobuf
    shellcheck
    shfmt
  ];
}
