package syslog

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"
)

//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=facility -output=facility_string.go
type facility byte

const (
	kern facility = iota
	user
	mail
	daemon
	auth
	syslog
	lpr
	news
	uucp
	clock
	authpriv
	ftp
	ntp
	audit
	alert
	cron
	local0
	local1
	local2
	local3
	local4
	local5
	local6
	local7
)

//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=severity -output=severity_string.go
type severity byte

const (
	EMERG severity = iota
	ALERT
	CRIT
	ERR
	WARNING
	NOTICE
	INFO
	DEBUG
)

type message struct {
	buf  [2048]byte
	size int
	time time.Time
	host net.IP

	// parsed fields
	priority byte
	hostname []byte
	app      []byte
	procid   []byte
	msgid    []byte
	msg      []byte
}

func (m *message) Facility() facility {
	return facility(m.priority / 8)
}

func (m *message) Host() string {
	return m.host.String()
}

func (m *message) Severity() severity {
	return severity(m.priority % 8)
}

var msgCleanup = strings.NewReplacer([]string{"\b", ""}...)

func (m *message) String() string {
	if m.msg == nil {
		return fmt.Sprintf("host=%s syslog=%q", m.host, m.buf[:m.size])
	}

	fields := make([]string, 0, 7)

	// fields = append(fields, fmt.Sprintf("ptr=%p", m))
	// fields = append(fields, "time=" + m.time.Format(time.RFC3339))

	if m.hostname != nil {
		fields = append(fields, fmt.Sprintf("host=%s", m.hostname))
	} else {
		fields = append(fields, "host="+m.host.String())
	}

	fields = append(fields, "facility="+m.Facility().String())
	fields = append(fields, "severity="+m.Severity().String())

	if m.app != nil {
		fields = append(fields, fmt.Sprintf("app-name=%s", m.app))
	}

	if m.procid != nil {
		fields = append(fields, fmt.Sprintf("procid=%s", m.procid))
	}

	if m.msgid != nil {
		fields = append(fields, fmt.Sprintf("msgid=%s", m.msgid))
	}

	fields = append(fields, fmt.Sprintf("msg=%q", msgCleanup.Replace(string(m.msg))))

	return strings.Join(fields, " ")
}

func (m *message) Timestamp() time.Time {
	return m.time
}

func (m *message) correctLegacyTime(t time.Time) {
	t = t.AddDate(m.time.Year(), 0, 0)

	offset := m.time.Sub(t) //nolint:ifshort // erroneous warning. offset is used below
	if offset < 0 {
		offset = -offset
	}

	if hoursOff := (offset - (offset % time.Hour)) / time.Hour; hoursOff > 1 {
		t = t.Add(hoursOff)
	}

	m.time = t
}

func (m *message) parse() bool {
	if !m.parsePriority() {
		return false
	}
	if !m.parseVersion() {
		return m.parseLegacyHeader()
	}
	if !m.parseHeader() {
		return false
	}

	return m.parseStructuredData()
}

func (m *message) parseHeader() bool {
	// TIMESTAMP HOSTNAME APP-NAME PROCID MSGID MSG
	parts := bytes.SplitN(m.msg, []byte{' '}, 6)

	if len(parts) != 6 || !m.parseTimestamp(parts[0]) {
		return false
	}
	m.hostname = ignoreNil(parts[1])
	m.app = ignoreNil(parts[2])
	m.procid = ignoreNil(parts[3])
	m.msgid = ignoreNil(parts[4])
	m.msg = parts[5]

	return true
}

func (m *message) parseStructuredData() bool {
	if len(m.msg) >= 2 && m.msg[0] == '-' && m.msg[1] == ' ' {
		m.msg = m.msg[2:]

		return true
	}

	return false
}

func (m *message) parseLegacyHeader() bool {
	const (
		layout  = time.Stamp
		timeLen = len(layout)
	)
	if len(m.msg) <= timeLen || m.msg[timeLen] != ' ' {
		goto parseHostname // too short or missing expected space after timestamp
	}

	if t, err := time.Parse(layout, string(m.msg[:timeLen])); err != nil {
		goto parseHostname // doesn't match the expected layout
	} else if !t.IsZero() { // if zero, ignore and use the current time
		m.correctLegacyTime(t)
	}
	m.msg = m.msg[timeLen+1:]

parseHostname:
	m.hostname = nil

	m.parseLegacyTag()

	m.trimSeverityPrefix()
	m.trimTimePrefix()
	m.trimCarriageReturns()

	return true
}

func (m *message) parseLegacyTag() {
	b := m.msg

	for i, c := range b {
		if c >= '0' && c <= '9' {
			continue
		}
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= 'A' && c <= 'Z' {
			continue
		}
		if c == '-' || c == '_' || c == '/' || c == '.' {
			continue
		}
		if c == '[' {
			m.app, b = b[:i], b[i:]

			goto parsePid
		}
		m.app, b = b[:i], b[i:]

		goto trimColon
	}
	m.app = nil
	m.procid = nil

	return

parsePid:
	if i := bytes.IndexByte(b[1:], ']'); i != -1 {
		m.procid = b[1 : 1+i]
		b = b[1+i+1:]
	} else {
		m.procid = nil
	}

trimColon:
	m.msg = bytes.TrimPrefix(b, []byte{':', ' '})
}

func (m *message) parsePriority() bool {
	if m.size < 3 || m.buf[0] != '<' {
		return false
	}
	var pri byte
	for i, c := range m.buf[1:5] {
		if c == '>' {
			m.priority = pri
			m.msg = m.buf[1+i+1 : m.size]

			return true
		}
		if c < '0' || c > '9' {
			return false
		}
		pri = pri*10 + c - '0'
	}

	return false
}

func (m *message) parseTimestamp(b []byte) bool {
	if ignoreNil(b) == nil {
		return true // NILVALUE
	}

	const (
		layout  = "2006-01-02T15:04:05.999999Z07:00"
		timeLen = len(layout)
	)
	if len(b) > timeLen {
		return false // too long
	}
	t, err := time.Parse(layout, string(b))
	if err != nil {
		return false
	}
	m.time = t

	return true
}

func (m *message) parseVersion() bool {
	if len(m.msg) < 2 {
		return false // too short
	}
	if m.msg[1] != ' ' {
		return false // missing space after version
	}
	if m.msg[0] != '1' {
		return false // we only support version 1
	}
	m.msg = m.msg[2:]

	return true
}

func (m *message) reset() {
	m.priority = 0
	m.hostname = nil
	m.app = nil
	m.procid = nil
	m.msgid = nil
	m.msg = nil
}

func (m *message) trimSeverityPrefix() {
	prefix := []byte(m.Severity().String() + ": ")
	m.msg = bytes.TrimPrefix(m.msg, prefix)
}

func (m *message) trimTimePrefix() {
	m.msg = bytes.TrimPrefix(m.msg, []byte(m.time.Format("2006-01-02 15:04:05 ")))
}

func (m *message) trimCarriageReturns() {
	if len(m.msg) > 0 && m.msg[0] == '\r' {
		m.msg = m.msg[1:]
	}
	// m.msg = bytes.Replace(m.msg, "\r", "(CR)", -1)
}

func ignoreNil(b []byte) []byte {
	if len(b) == 1 && b[0] == '-' {
		return nil
	}

	return b
}
