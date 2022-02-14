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

type BootScript func(context.Context, Job, *ipxe.Script)

type BootScripter interface {
	BootScript(string) BootScript
}

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
	span.SetAttributes(attribute.String("boots.script_name", name))

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
	s.Set("iface", j.InterfaceName(0))
	s.Or("shell")
	s.Set("tinkerbell", "http://"+conf.PublicFQDN)
	s.Set("syslog_host", conf.PublicSyslogFQDN)
	s.Set("ipxe_cloud_config", "packet")

	// the trace id is enough to find otel traces in most systems
	if sc := span.SpanContext(); sc.IsSampled() {
		s.Echo("Debug Trace ID: " + sc.TraceID().String())
	}

	fn(ctx, j, s)
	script := s.Bytes()
	span.SetAttributes(attribute.String("ipxe-script", string(script)))

	if _, err := w.Write(script); err != nil {
		j.With("script", name).Error(errors.Wrap(err, "unable to write boot script"))
		span.SetStatus(codes.Error, err.Error())

		return
	}
}

func (i Installers) auto(ctx context.Context, j Job, s *ipxe.Script) {
	if j.instance == nil {
		j.Info(errors.New("no device to boot, providing an iPXE shell"))
		shell(ctx, j, s)

		return
	}
	if f, ok := i.ByInstaller[j.hardware.OperatingSystem().Installer]; ok {
		f(ctx, j, s)

		return
	}
	if f, ok := i.BySlug[j.hardware.OperatingSystem().Slug]; ok {
		f(ctx, j, s)

		return
	}
	if f, ok := i.ByDistro[j.hardware.OperatingSystem().Distro]; ok {
		f(ctx, j, s)

		return
	}
	if i.Default != nil {
		i.Default(ctx, j, s)

		return
	}
	j.With("slug", j.hardware.OperatingSystem().Slug, "distro", j.hardware.OperatingSystem().Distro).Error(errors.New("unsupported slug/distro"))
	shell(ctx, j, s)
}

func shell(ctx context.Context, j Job, s *ipxe.Script) {
	s.Shell()
}
