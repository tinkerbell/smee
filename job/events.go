package job

// TODO(SWE-338) move to separate package, define consts for strings like provisioning.104.01

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/tinkerbell/boots/packet"
	"github.com/pkg/errors"
)

func (j Job) CustomPXEDone() {
	if j.InstanceID() == "" {
		j.Info("CustomPXEDone called for nil instance")
		return
	}

	// We close the job here since we have no visibility beyond this point.
	j.With("mode", j.mode).Info("detected a finished custom_ipxe")

	e := event{_kind: "phone-home"}

	if err := e.postInstance(j.instance.ID); err != nil {
		j.With("os", "custom_ipxe").Error(errors.WithMessage(err, "posting phone-home event"))
	}
}

func (j Job) DisablePXE() {
	if j.instance == nil {
		j.Error(errors.New("instance is nil"))
		return
	}

	if err := client.UpdateInstance(j.instance.ID, strings.NewReader(`{"allow_pxe":false}`)); err != nil {
		j.Error(errors.WithMessage(err, "disabling PXE"))
		return
	}

	j.With("allow_pxe", false).Info("updated allow_pxe")
}

func (j Job) PostHardwareProblem(slug string) bool {
	if j.hardware == nil {
		return false
	}
	var v struct {
		Problem string `json:"problem"`
	}
	v.Problem = slug
	b, err := json.Marshal(&v)
	if err != nil {
		j.With("problem", slug).Error(errors.WithMessage(err, "encoding hardware problem request"))
		return false
	}
	if _, err := client.PostHardwareProblem(j.hardware.ID, bytes.NewReader(b)); err != nil {
		j.With("problem", slug).Error(errors.WithMessage(err, "posting hardware problem"))
		return false
	}
	return true
}

type poster interface {
	postInstance(string) error
	postHardware(string) error
	kind() string
	id() string
}

func (j Job) phoneHome(body []byte) bool {
	p, err := posterFromJSON(body)
	if err != nil {
		j.Error(errors.WithMessage(err, "parsing event"))
		return false
	}

	var id string
	var typ string
	var post func(string) error
	var disablePXE bool
	if j.InstanceID() != "" {
		id = j.instance.ID
		typ = "instance"
		post = p.postInstance
		if p.kind() == "provisioning.104.01" {
			disablePXE = true
			if j.instance.OS.OsSlug == "custom_ipxe" {
				defer j.CustomPXEDone()
			}
		}
	} else {
		if j.HardwareState() != "preinstalling" {
			j.With("state", j.HardwareState()).Info("ignoring hardware phone-home when state is not preinstalling")
			return false
		}
		id = j.hardware.ID
		typ = "hardware"
		post = p.postHardware
	}

	if err := post(id); err != nil {
		j.With("kind", p.kind(), "type", typ).Error(err)
		return false
	}

	if p.id() != "" {
		j.With("kind", p.kind(), "id", p.id()).Info("proxied event")
	} else {
		j.With("kind", p.kind()).Info("proxied event")
	}

	if disablePXE {
		j.DisablePXE()
	}

	return true
}
func (j Job) postEvent(kind, body string, private bool) bool {
	if j.InstanceID() == "" {
		j.With("kind", kind).Error(errors.New("postEvent called for nil instance"))
		return false
	}
	e, err := newEvent(kind, body, private)
	if err != nil {
		j.With("kind", kind).Error(errors.WithMessage(err, "encoding event"))
		return false
	}
	if err := e.postInstance(j.instance.ID); err != nil {
		// do not use j.Error to avoid infinite recursion
		j.With("kind", kind).Error(err, "posting event")
	}
	if e.id() != "" {
		j.With("kind", e.kind(), "id", e.id()).Info("posted event")
	} else {
		j.With("kind", e.kind()).Info("posted event")
	}
	return true
}

// golangci-lint: unused
//func (j Job) postFailure(reason string) bool {
//	f := failure{
//		Reason: reason,
//	}
//
//	if j.InstanceID() != "" {
//		if err := f.postInstance(j.instance.ID); err != nil {
//			j.Error(errors.WithMessage(err, "posting instance failure"))
//			return false
//		}
//	} else {
//		if err := f.postHardware(j.hardware.ID); err != nil {
//			j.Error(errors.WithMessage(err, "posting hardware failure"))
//			return false
//		}
//	}
//	return true
//}

func posterFromJSON(b []byte) (poster, error) {
	if len(b) == 0 {
		return &event{_kind: "phone-home"}, nil
	}
	var res struct {
		Type     string `json:"type"`
		Password []byte `json:"password"`
		Instance string `json:"instance_id,omitempty"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return &event{}, errors.Wrap(err, "unmarshalling event body")
	}
	if res.Type == "" {
		if len(res.Instance) > 0 {
			return &event{_kind: "phone-home"}, nil
		}
		if len(res.Password) > 0 {
			pass, err := decryptPassword(res.Password)
			if err != nil {
				return &event{}, err
			}
			return &event{_kind: "phone-home", pass: pass}, nil
		}
	}
	if res.Type == "failure" {
		var f failure
		if err := json.Unmarshal(b, &f); err != nil {
			return nil, errors.Wrap(err, "unmarshalling failure body")
		}
		return &f, nil
	}
	return &event{_kind: res.Type, json: b}, nil
}

func newEvent(kind, body string, private bool) (event, error) {
	b, err := json.Marshal(&packet.Event{Type: kind, Body: body, Private: private})
	if err != nil {
		return event{}, errors.Wrap(err, "marshalling event")
	}
	return event{_kind: kind, json: b}, nil
}

type event struct {
	_id   string
	_kind string
	pass  string
	json  []byte
}

func (e *event) post(endpoint, id string) error {
	if id == "" {
		return errors.New("missing id")
	}

	if e.pass != "" {
		return client.PostInstancePassword(id, e.pass)
	}

	if endpoint == "hardware" {
		if e._kind == "phone-home" {
			return client.PostHardwarePhoneHome(id)
		} else {
			var err error
			e._id, err = client.PostHardwareEvent(id, bytes.NewReader(e.json))
			return err
		}
	} else if endpoint == "instance" {
		if e._kind == "phone-home" {
			return client.PostInstancePhoneHome(id)
		} else {
			var err error
			e._id, err = client.PostInstanceEvent(id, bytes.NewReader(e.json))
			return err
		}
	}
	return errors.New("unknown endpoint: " + endpoint)
}
func (e *event) postInstance(id string) (err error) {
	return e.post("instance", id)
}
func (e *event) postHardware(id string) (err error) {
	return e.post("hardware", id)
}
func (e *event) kind() string {
	return e._kind
}
func (e *event) id() string {
	return e._id
}

type failure struct {
	Reason  string `json:"reason"`
	Private bool   `json:"private"`
}

func (f *failure) post(typ, id string) error {

	if id == "" {
		return errors.New("missing id")
	}

	f.Private = true
	b, err := json.Marshal(f)
	if err != nil {
		return errors.Wrap(err, "marshalling failure event")
	}
	if typ == "hardware" {
		return client.PostHardwareFail(id, bytes.NewReader(b))
	} else if typ == "instance" {
		return client.PostInstanceFail(id, bytes.NewReader(b))
	}

	return errors.New("unknown type: " + typ)
}
func (f *failure) postInstance(id string) error {
	return f.post("instance", id)
}
func (f *failure) postHardware(id string) error {
	return f.post("hardware", id)
}
func (f *failure) kind() string {
	return "failure"
}
func (f *failure) id() string {
	return "no-id"
}
