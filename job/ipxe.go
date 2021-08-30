package job

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/ipxe"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type BootScript func(Job, ipxe.Script) ipxe.Script

func (i *Installers) RegisterDefaultInstaller(bs BootScript) {
	if i.Default != nil {
		err := errors.New("default installer already registered!")
		joblog.Fatal(err)
	}
	i.Default = bs
}

func (i *Installers) RegisterDistro(name string, builder BootScript) {
	if _, ok := i.ByDistro[name]; ok {
		err := errors.Errorf("distro %q already registered!", name)
		joblog.Fatal(err, "distro", name)
	}
	i.ByDistro[name] = builder
}

func (i *Installers) RegisterInstaller(name string, builder BootScript) {
	if _, ok := i.ByInstaller[name]; ok {
		err := errors.Errorf("installer %q already registered!", name)
		joblog.Fatal(err, "installer", name)
	}
	i.ByInstaller[name] = builder
}

func (i *Installers) RegisterSlug(name string, builder BootScript) {
	if _, ok := i.BySlug[name]; ok {
		err := errors.Errorf("slug %q already registered!", name)
		joblog.Fatal(err, "slug", name)
	}
	i.BySlug[name] = builder
}

func (j Job) serveBootScript(ctx context.Context, w http.ResponseWriter, name string, i Installers) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("name", name))

	scripts := map[string]BootScript{
		"auto":  i.auto,
		"shell": shell,
	}
	fn, ok := scripts[name]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		err := errors.Errorf("boot script %q not found", name)
		j.With("script", name).Error(err)
		span.SetStatus(codes.Error, err.Error())

		return
	}

	s := ipxe.NewScript()
	s.Set("iface", j.InterfaceName(0)).Or("shell")
	s.Set("tinkerbell", "http://"+conf.PublicFQDN)
	s.Set("syslog_host", conf.PublicSyslogFQDN)
	s.Set("ipxe_cloud_config", "packet")

	s.Echo("Packet.net Baremetal - iPXE boot")

	iScript := fn(j, *s)
	src := iScript.Bytes()
	span.SetAttributes(attribute.String("ipxe-script", string(src)))

	if _, err := w.Write(src); err != nil {
		j.With("script", name).Error(errors.Wrap(err, "unable to write boot script"))
		span.SetStatus(codes.Error, err.Error())

		return
	}
}

func (i Installers) auto(j Job, s ipxe.Script) ipxe.Script {
	if j.instance == nil {
		j.Info(errors.New("no device to boot, providing an iPXE shell"))

		return *s.Shell()
	}
	if f, ok := i.ByInstaller[j.hardware.OperatingSystem().Installer]; ok {
		f(j, s)

		return f(j, s)
	}
	if f, ok := i.BySlug[j.hardware.OperatingSystem().Slug]; ok {
		return f(j, s)
	}
	if f, ok := i.ByDistro[j.hardware.OperatingSystem().Distro]; ok {
		return f(j, s)
	}
	if i.Default != nil {
		return i.Default(j, s)
	}
	j.With("slug", j.hardware.OperatingSystem().Slug, "distro", j.hardware.OperatingSystem().Distro).Error(errors.New("unsupported slug/distro"))

	return *s.Shell()
}

func shell(j Job, s ipxe.Script) ipxe.Script {
	return *s.Shell()
}
