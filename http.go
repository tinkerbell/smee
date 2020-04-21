package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sebest/xff"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/httplog"
	"github.com/tinkerbell/boots/job"
)

var (
	httpAddr = conf.HTTPBind
)

func init() {
	flag.StringVar(&httpAddr, "http-addr", httpAddr, "IP and port to listen on for HTTP.")
}

func serveHealthchecker(rev string, start time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		res := struct {
			GitRev     string  `json:"git_rev"`
			Uptime     float64 `json:"uptime"`
			Goroutines int     `json:"goroutines"`
		}{
			GitRev:     rev,
			Uptime:     time.Since(start).Seconds(),
			Goroutines: runtime.NumGoroutine(),
		}

		b, err := json.Marshal(&res)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			mainlog.Error(errors.Wrap(err, "marshaling healtcheck json"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

// ServeHTTP is a useless comment
func ServeHTTP() {
	http.HandleFunc("/", serveJobFile)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/_packet/healthcheck", http.HandlerFunc(serveHealthchecker(GitRev, StartTime)))
	http.Handle("/healthcheck", http.HandlerFunc(serveHealthchecker(GitRev, StartTime)))
	http.HandleFunc("/phone-home", servePhoneHome)
	http.HandleFunc("/phone-home/key", job.ServePublicKey)
	http.HandleFunc("/problem", serveProblem)
	// Events endpoint used to forward customer generated custom events from a running device (instance) to packet API
	http.HandleFunc("/events", func(w http.ResponseWriter, req *http.Request) {
		code, err := serveEvents(client, w, req)
		if err == nil {
			return
		}
		if code != http.StatusOK {
			mainlog.Error(err)
		}
	})
	http.HandleFunc("/hardware-components", serveHardware)

	var h http.Handler
	if len(conf.TrustedProxies) > 0 {
		xffmw, _ := xff.New(xff.Options{
			AllowedSubnets: conf.TrustedProxies,
		})

		h = xffmw.Handler(&httplog.Handler{
			Handler: http.DefaultServeMux,
		})
	} else {
		h = &httplog.Handler{
			Handler: http.DefaultServeMux,
		}
	}

	if err := http.ListenAndServe(httpAddr, h); err != nil {
		err = errors.Wrap(err, "listen and serve http")
		mainlog.Fatal(err)
	}
}

func serveJobFile(w http.ResponseWriter, req *http.Request) {
	j, err := job.CreateFromRemoteAddr(req.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		mainlog.With("client", req.RemoteAddr, "error", err).Info("no job found for client address")
		return
	}
	j.ServeFile(w, req)
}

func serveHardware(w http.ResponseWriter, req *http.Request) {
	j, err := job.CreateFromRemoteAddr(req.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		mainlog.With("client", req.RemoteAddr, "error", err).Info("no job found for client address")
		return
	}
	j.AddHardware(w, req)
}

func servePhoneHome(w http.ResponseWriter, req *http.Request) {
	j, err := job.CreateFromRemoteAddr(req.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		mainlog.With("client", req.RemoteAddr, "error", err).Info("no job found for client address")
		return
	}
	j.ServePhoneHomeEndpoint(w, req)
}

func serveProblem(w http.ResponseWriter, req *http.Request) {
	j, err := job.CreateFromRemoteAddr(req.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		mainlog.With("client", req.RemoteAddr, "error", err).Info("no job found for client address")
		return
	}
	j.ServeProblemEndpoint(w, req)
}

func readClose(r io.ReadCloser) (b []byte, err error) {
	b, err = ioutil.ReadAll(r)
	err = errors.Wrap(err, "read data")
	r.Close()
	return
}

type eventsServer interface {
	GetInstanceIDFromIP(net.IP) (string, error)
	PostInstanceEvent(string, io.Reader) (string, error)
}

// Forward user generated events to Packet API
func serveEvents(client eventsServer, w http.ResponseWriter, req *http.Request) (int, error) {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return http.StatusBadRequest, errors.Wrap(err, "split host port")
	}

	ip := net.ParseIP(host)
	if ip == nil {
		w.WriteHeader(http.StatusOK)
		return http.StatusOK, errors.New("no device found for client address")
	}

	deviceID, err := client.GetInstanceIDFromIP(ip)
	if err != nil || deviceID == "" {
		w.WriteHeader(http.StatusOK)
		return http.StatusOK, errors.New("no device found for client address")
	}

	b, err := readClose(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return http.StatusBadRequest, err
	}
	if len(b) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return http.StatusBadRequest, errors.New("userEvent body is empty")
	}

	var res struct {
		Code    int    `json:"code"`
		State   string `json:"state"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return http.StatusBadRequest, errors.New("userEvent cannot be generated from supplied json")
	}

	e := struct {
		Code    string `json:"type"`
		State   string `json:"state"`
		Message string `json:"body"`
	}{
		Code:    "user." + strconv.Itoa(res.Code),
		State:   res.State,
		Message: res.Message,
	}
	payload, err := json.Marshal(e)
	if err != nil {
		// TODO(mmlb): this should be 500
		w.WriteHeader(http.StatusBadRequest)
		return http.StatusBadRequest, errors.New("userEvent cannot be encoded")
	}

	if _, err := client.PostInstanceEvent(deviceID, bytes.NewReader(payload)); err != nil {
		// TODO(mmlb): this should be 500
		w.WriteHeader(http.StatusBadRequest)
		return http.StatusBadRequest, errors.New("failed to post userEvent")
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
	return http.StatusOK, nil
}
