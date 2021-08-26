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

var (
	byDistro         = make(map[string]BootScript)
	byInstaller      = make(map[string]BootScript)
	bySlug           = make(map[string]BootScript)
	defaultInstaller BootScript
	scripts          = map[string]BootScript{
		"auto":  auto,
		"shell": shell,
	}
)

type BootScript func(context.Context, Job, *ipxe.Script)

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

func RegisterInstaller(name string, builder BootScript) {
	if _, ok := byInstaller[name]; ok {
		err := errors.Errorf("installer %q already registered!", name)
		joblog.Fatal(err, "installer", name)
	}
	byInstaller[name] = builder
}

func RegisterSlug(name string, builder BootScript) {
	if _, ok := bySlug[name]; ok {
		err := errors.Errorf("slug %q already registered!", name)
		joblog.Fatal(err, "slug", name)
	}
	bySlug[name] = builder
}

func (j Job) serveBootScript(ctx context.Context, w http.ResponseWriter, name string) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("boots.script_name", name))

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

	// the trace id is enough to find otel traces in most systems
	s.Echo("Debug Trace ID: " + span.SpanContext().TraceID().String())

	fn(ctx, j, s)
	src := s.Bytes()
	span.SetAttributes(attribute.String("ipxe-script", string(src)))

	if _, err := w.Write(src); err != nil {
		j.With("script", name).Error(errors.Wrap(err, "unable to write boot script"))
		span.SetStatus(codes.Error, err.Error())

		return
	}
}

func auto(ctx context.Context, j Job, s *ipxe.Script) {
	if j.instance == nil {
		j.Info(errors.New("no device to boot, providing an iPXE shell"))
		shell(ctx, j, s)

		return
	}
	if f, ok := byInstaller[j.hardware.OperatingSystem().Installer]; ok {
		f(ctx, j, s)

		return
	}
	if f, ok := bySlug[j.hardware.OperatingSystem().Slug]; ok {
		f(ctx, j, s)

		return
	}
	if f, ok := byDistro[j.hardware.OperatingSystem().Distro]; ok {
		f(ctx, j, s)

		return
	}
	if defaultInstaller != nil {
		defaultInstaller(ctx, j, s)

		return
	}
	j.With("slug", j.hardware.OperatingSystem().Slug, "distro", j.hardware.OperatingSystem().Distro).Error(errors.New("unsupported slug/distro"))
	shell(ctx, j, s)
}

func shell(ctx context.Context, j Job, s *ipxe.Script) {
	s.Shell()
}
