package syslog

import (
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

var syslogMessagePool = sync.Pool{
	New: func() interface{} { return new(message) },
}

type Receiver struct {
	c     *net.UDPConn
	parse chan *message
	done  chan struct{}
	err   error

	Logger logr.Logger
}

func StartReceiver(laddr string, parsers int) (*Receiver, error) {
	if parsers < 1 {
		parsers = 1
	}

	addr, err := net.ResolveUDPAddr("udp4", laddr)
	if err != nil {
		return nil, errors.Wrap(err, "resolve syslog udp listen address")
	}

	c, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, errors.Wrap(err, "listen on syslog udp address")
	}

	s := &Receiver{
		c:     c,
		parse: make(chan *message, parsers),
		done:  make(chan struct{}),
	}

	for i := 0; i < parsers; i++ {
		go s.runParser()
	}
	go s.run()

	return s, nil
}

func (r *Receiver) Done() <-chan struct{} {
	return r.done
}

func (r *Receiver) Err() error {
	return r.err
}

func (r *Receiver) cleanup() {
	r.c.Close()

	close(r.parse)
	close(r.done)
}

func (r *Receiver) run() {
	var msg *message
	defer func() {
		if msg != nil {
			syslogMessagePool.Put(msg)
		}
		r.cleanup()
	}()

	for {
		if msg == nil {
			var ok bool
			msg, ok = syslogMessagePool.Get().(*message)
			if !ok {
				r.Logger.Error(errors.New("error type asserting pool item into message"), "error type asserting pool item into message")

				continue
			}
		}
		n, from, err := r.c.ReadFromUDP(msg.buf[:])
		if err != nil {
			err = errors.Wrap(err, "error reading udp message")
			if _, ok := err.(net.Error); ok {
				r.Logger.Error(err, "error reading udp message")

				continue
			}
			r.err = err

			return
		}
		msg.time = time.Now().UTC()
		msg.host = from.IP
		msg.size = n
		r.parse <- msg
		msg = nil
	}
}

func parse(m *message) map[string]interface{} {
	structured := make(map[string]interface{})
	if m.Facility().String() != "" {
		structured["facility"] = m.Facility().String()
	}
	if m.Severity().String() != "" {
		structured["severity"] = m.Severity().String()
	}
	if string(m.hostname) != "" {
		structured["hostname"] = string(m.hostname)
	}
	if string(m.app) != "" {
		structured["app-name"] = string(m.app)
	}
	if string(m.procid) != "" {
		structured["procid"] = string(m.procid)
	}
	if string(m.msgid) != "" {
		structured["msgid"] = string(m.msgid)
	}
	if string(m.msg) != "" {
		if strings.HasPrefix(string(m.msg), "{") {
			var j map[string]interface{}
			if err := json.Unmarshal(m.msg, &j); err == nil {
				structured["msg"] = j
			}
		} else {
			structured["msg"] = string(m.msg)
		}
	}
	structured["host"] = m.host.String()

	return structured
}

func (r *Receiver) runParser() {
	for m := range r.parse {
		if m.parse() {
			structured := parse(m)
			sl := r.Logger.WithValues("msg", structured)
			if m.Severity() == DEBUG {
				sl.V(1).Info("msg", "msg", m)
			} else {
				sl.Info("msg", "msg", m)
			}
		} else {
			r.Logger.V(1).Info("msg", "msg", m)
		}
		m.reset()
		syslogMessagePool.Put(m)
	}
}
