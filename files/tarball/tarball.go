package tarball

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
)

type File struct {
	t *Tarball
}

func (f *File) Close() (err error) {
	err = f.t.flush()
	f.t = nil

	return
}

func (f File) Write(b []byte) (int, error) {
	return f.t.buf.Write(b)
}

func (f File) Writef(format string, args ...interface{}) {
	if len(args) == 0 {
		f.t.buf.WriteString(format)
	} else {
		fmt.Fprintf(&f.t.buf, format, args...)
	}
}

func (f File) WriteString(s string) (int, error) {
	l, err := f.t.buf.WriteString(s)

	return l, errors.Wrap(err, "write string to file")
}

type Tarball struct {
	gz *gzip.Writer
	tw *tar.Writer

	hdr tar.Header
	buf bytes.Buffer
}

func New(w io.Writer) *Tarball {
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)

	return &Tarball{
		gz: gz,
		tw: tw,
	}
}

func (t *Tarball) NewFile(name string, mode int64, kind byte) File {
	t.hdr = tar.Header{
		Name:     name,
		Mode:     mode,
		ModTime:  time.Now(),
		Typeflag: kind,
	}
	t.buf.Reset()

	return File{t}
}

func (t *Tarball) Close() error {
	errTar := t.tw.Close()
	errGZ := t.gz.Close()

	if errTar != nil {
		return errors.Wrap(errTar, "flushing and closing tar writer")
	}

	return errors.Wrap(errGZ, "flushing and closing gzip writer")
}

func (t *Tarball) flush() error {
	if t == nil {
		return nil
	}

	t.hdr.Size = int64(t.buf.Len())
	t.tw.WriteHeader(&t.hdr)

	_, err := t.tw.Write(t.buf.Bytes())

	return errors.Wrap(err, "flush tarball")
}
