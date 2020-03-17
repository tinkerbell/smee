package job

import (
	"net/http"

	"github.com/packethost/tinkerbell/env"
	"github.com/packethost/tinkerbell/ipxe"
	"github.com/pkg/errors"
)

var (
	byDistro         = make(map[string]BootScript)
	bySlug           = make(map[string]BootScript)
	defaultInstaller BootScript
	scripts          = map[string]BootScript{
		"auto":  auto,
		"shell": shell,
	}
)

type BootScript func(Job, *ipxe.Script)

func RegisterDefaultInstaller(bootScript BootScript) {
	if defaultInstaller != nil {
		err := errors.New("default installer already registered!")
		joblog.Fatal(err)
	}
	defaultInstaller = bootScript
}

func RegisterDistro(name string, builder BootScript) {
	if _, ok := byDistro[name]; ok {
		err := errors.Errorf("distro %q already registered!", name)
		joblog.Fatal(err, "distro", name)
	}
	byDistro[name] = builder
}

func RegisterSlug(name string, builder BootScript) {
	if _, ok := bySlug[name]; ok {
		err := errors.Errorf("slug %q already registered!", name)
		joblog.Fatal(err, "slug", name)
	}
	bySlug[name] = builder
}

func (j Job) serveBootScript(w http.ResponseWriter, req *http.Request, name string) {
	fn, ok := scripts[name]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		j.With("script", name).Error(errors.New("boot script not found"))
		return
	}

	s := ipxe.NewScript()
	s.Set("iface", j.InterfaceName(0)).Or("shell")
	s.Set("tinkerbell", "http://"+env.PublicFQDN)
	s.Set("ipxe_cloud_config", "packet")

	s.Echo("Packet.net Baremetal - iPXE boot")

	fn(j, s)

	if _, err := w.Write(s.Bytes()); err != nil {
		j.With("script", name).Error(errors.Wrap(err, "unable to write boot script"))
		return
	}
}

func auto(j Job, s *ipxe.Script) {
	if j.instance == nil {
		j.Info(errors.New("no device to boot, providing an iPXE shell"))
		shell(j, s)
		return
	}
	if f, ok := bySlug[j.instance.OS.Slug]; ok {
		f(j, s)
		return
	}
	if f, ok := byDistro[j.instance.OS.Distro]; ok {
		f(j, s)
		return
	}
	if defaultInstaller != nil {
		defaultInstaller(j, s)
		return
	}
	j.With("slug", j.instance.OS.Slug, "distro", j.instance.OS.Distro).Error(errors.New("unsupported slug/distro"))
	shell(j, s)
}

func shell(j Job, s *ipxe.Script) {
	s.Shell()
}
