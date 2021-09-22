package nixos

import (
	"context"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

type Installer struct {
	Paths map[string]string
}

func BuildInitPaths() map[string]string {
	paths := map[string]string{}

	for _, env := range os.Environ() {
		kv := strings.Split(env, "=")
		if len(kv) != 2 {
			continue
		}

		k, v := kv[0], kv[1]
		k = strings.ToLower(k)
		if !strings.HasPrefix(k, "nixos_") {
			continue
		}

		// shell env vars are only [a-zA-Z0-9_] so we use "__" to separate os and hw slugs
		slugs := strings.Split(k, "__")
		if len(slugs) != 2 {
			continue
		}

		// shell env vars are only [a-zA-Z0-9_] so we use "_" in place of "." in hw slug, so need to go back
		os, hw := slugs[0], slugs[1]
		hw = strings.Replace(hw, "_", ".", -1)
		k = os + "/" + hw

		paths[k] = "/nix/store/" + v + "/init"

	}

	return paths
}

func (i Installer) BootScript() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		key := j.OperatingSystem().Slug + "/" + j.PlanSlug()
		init := i.Paths[key]
		if init == "" {
			tag := j.OperatingSystem().ImageTag
			if tag == "" {
				j.With("slug", j.OperatingSystem().Slug, "class", j.PlanSlug()).Error(errors.New("unknown os/class combo and no OSV ImageTag set"))

				return *s.Shell()
			}
			key = j.OperatingSystem().Slug + "/" + tag
			init = "/nix/store/" + tag + "/init"
		}

		s.PhoneHome("provisioning.104.01")
		s.Set("base-url", conf.MirrorBase+"/misc/tinkerbell/nixos/"+key)
		s.Kernel("${base-url}/kernel")
		ks := kernelParams(j, s, init)

		ks.Initrd("${base-url}/initrd")
		ks.Boot()

		return ks
	}
}

func kernelParams(j job.Job, s ipxe.Script, init string) ipxe.Script {
	s.Args("init=" + init)
	s.Args("initrd=initrd")
	if j.IsARM() {
		s.Args("cma=0M")
		s.Args("biosdevname=0")
		s.Args("net.ifnames=0")
		s.Args("console=ttyAMA0,115200")
	} else {
		s.Args("console=ttyS1,115200")
	}
	s.Args("loglevel=7")
	if j.PasswordHash() != "" {
		s.Args("pwhash=" + j.PasswordHash())
	}

	return s
}
