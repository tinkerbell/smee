package job

import (
	"net/url"
	"strings"
)

type Mode uint32

const (
	modeNone Mode = iota
	modeHardware
	modeManagement
	modeInstance
	modeProv
	modeDeprov
)

var modeSlugs = map[Mode]string{
	modeNone:       "none",
	modeHardware:   "hardware",
	modeManagement: "management",
	modeInstance:   "instance",
	modeProv:       "prov",
	modeDeprov:     "deprov",
}

func (m Mode) Slug() string {
	if s, ok := modeSlugs[m]; ok {
		return s
	}

	return "unknown"
}

var modeStrings = map[Mode]string{
	modeNone:       "(no mode)",
	modeHardware:   "Hardware",
	modeManagement: "Management",
	modeInstance:   "Instance",
	modeProv:       "Provision",
	modeDeprov:     "Deprovision",
}

func (m Mode) String() string {
	if s, ok := modeStrings[m]; ok {
		return s
	}

	return "(unknown mode)"
}

var modesBySlug = func() map[string]Mode {
	modes := make(map[string]Mode, len(modeSlugs))
	for mode, slug := range modeSlugs {
		modes[slug] = mode
	}

	return modes
}()

func modesFromQuery(params url.Values) map[Mode]bool {
	modes := make(map[Mode]bool, len(modeSlugs))

	str := params.Get("modes")
	if str == "" {
		str = params.Get("mode") // Because even I make mistakes!
	}
	for _, slug := range strings.Split(str, ",") {
		if mode, ok := modesBySlug[slug]; ok {
			modes[mode] = true
		}
	}

	return modes
}
