# Tinkerbell
[![Build Status][boots_ci_status]][boots_ci]
[![codecov](https://codecov.io/gh/packethost/boots/branch/master/graph/badge.svg?token=JH41dqSgYI)](https://codecov.io/gh/packethost/boots)

This services handles PXE and DHCP for provisions

### Local Setup

First, you need to make sure you have [git-lfs](https://git-lfs.github.com/) installed

```
git lfs install
git lfs pull
```

Running the Tests
```
# ensure you have the right packages
dep ensure
# make the files
make all
# run the tests
go test
```

Build/Run Tinkerbell
```
# run boots
./boots
```

You can use NixOS shell, which will have the Git-LFS, Go and others

`nix-shell`

Note: for mac users, you will need to comment out the line `pkgsCross.aarch64-multiplatform.buildPackages.gcc` in order for the build to work

[boots_ci]: https://drone.packet.net/packethost/boots
[boots_ci_status]: https://drone.packet.net/api/badges/packethost/boots/status.svg
