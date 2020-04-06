# Boots
[![Build Status](https://cloud.drone.io/api/badges/tinkerbell/boots/status.svg)](https://cloud.drone.io/tinkerbell/boots)

This services handles DHCP, PXE, tftp, and iPXE for provisions.

### Local Setup

First, you need to make sure you have [git-lfs](https://github.com/git-lfs/git-lfs/wiki/Installation) installed:

```
# install "git-lfs" package for your OS, i.e.:
curl -s https://packagecloud.io/install/repositories/github/git-lfs/script.deb.sh | sudo bash
apt install git-lfs

# then run these two commands:
git lfs install
git lfs pull
```

Running the Tests
```
# make the files
make all
# run the tests
go test
```

Build/Run Boots
```
# run boots
./boots
```

You can use NixOS shell, which will have the Git-LFS, Go and others

`nix-shell`

Note: for mac users, you will need to comment out the line `pkgsCross.aarch64-multiplatform.buildPackages.gcc` in order for the build to work

