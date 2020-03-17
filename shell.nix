let _pkgs = import <nixpkgs> { };
in { pkgs ? import (_pkgs.fetchFromGitHub {
    owner = "NixOS";
    repo = "nixpkgs-channels";
    # nixos-unstable @2019-05-30
    rev = "a02dfde07417ead2ab9b24443f195dc8532b409c";
    sha256 = "0g201slnc2f5w7k7xqzc0s3q1ckfg8xqb40hamhzl9a4vd1hvbwj";
  }) { } }:

with pkgs;

mkShell {
  buildInputs = [
    dep
    gcc
    git-lfs
    gnumake
    go
    pkgsCross.aarch64-multiplatform.buildPackages.gcc
    xz
  ];
}
