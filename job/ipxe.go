package job

import (
	"context"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/ipxe"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (j *Job) serveBootScript(ctx context.Context, w http.ResponseWriter, name string) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("boots.script_name", name))
	var script []byte
	switch name {
	case "auto":
		if cs, err := j.customScript(); err == nil {
			script = []byte(cs)
			break
		}
		s, err := j.defaultScript(span)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			err := errors.Errorf("boot script %q not found", name)
			j.Logger.Error(err, "error", "script", name)
			span.SetStatus(codes.Error, err.Error())

			return
		}
		script = []byte(s)
	default:
		w.WriteHeader(http.StatusNotFound)
		err := errors.Errorf("boot script %q not found", name)
		j.Logger.Error(err, "error", "script", name)
		span.SetStatus(codes.Error, err.Error())

		return
	}
	span.SetAttributes(attribute.String("ipxe-script", string(script)))

	if _, err := w.Write(script); err != nil {
		j.Logger.Error(errors.Wrap(err, "unable to write boot script"), "unable to write boot script", "script", name)
		span.SetStatus(codes.Error, err.Error())

		return
	}
}

// osieDownloadURL returns the value of Custom OSIE Service Version or just /current.
func (j *Job) osieDownloadURL(osieURL string, osieFullURLOverride string) string {
	if osieFullURLOverride != "" {
		return osieFullURLOverride
	}
	if u := j.OSIEBaseURL(); u != "" {
		return u
	}
	if j.OSIEVersion() != "" {
		return osieURL + "/" + j.OSIEVersion()
	}

	return osieURL + "/current"
}

func (j *Job) defaultScript(span trace.Span) (string, error) {
	auto := ipxe.Hook{
		Arch:              j.Arch(),
		Console:           "",
		DownloadURL:       j.osieDownloadURL(conf.MirrorBaseURL+"/misc/osie", j.OSIEURLOverride),
		ExtraKernelParams: j.ExtraKernelParams,
		Facility:          j.FacilityCode(),
		HWAddr:            j.PrimaryNIC().String(),
		SyslogHost:        conf.PublicSyslogFQDN,
		TinkerbellTLS:     j.TinkServerTLS,
		TinkGRPCAuthority: j.TinkServerGRPCAddr,
		VLANID:            j.VLANID(),
		WorkerID:          j.InstanceID(),
	}
	if sc := span.SpanContext(); sc.IsSampled() {
		auto.TraceID = sc.TraceID().String()
	}

	return ipxe.GenerateTemplate(auto, ipxe.HookScript)
}

func (j *Job) customScript() (string, error) {
	if chain := j.hardware.IPXEURL(j.mac); chain != "" {
		u, err := url.Parse(chain)
		if err != nil {
			return "", errors.Wrap(err, "invalid custom chain URL")
		}
		c := ipxe.Custom{Chain: u}
		return ipxe.GenerateTemplate(c, ipxe.CustomScript)
	}
	if script := j.hardware.IPXEScript(j.mac); script != "" {
		c := ipxe.Custom{Script: script}
		return ipxe.GenerateTemplate(c, ipxe.CustomScript)
	}

	return "", errors.New("no custom script or chain defined in the hardware data")
}
