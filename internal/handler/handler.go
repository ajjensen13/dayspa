package handler

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	webRootFlag        = `web-root`
	webRootShortHand   = `w`
	webRootDescription = "root directory from which to serve web files"
	urlPrefixFlag      = `url-prefix`
	urlPrefixDefault   = `/`
)

func Init(cmd *cobra.Command) {
	cmd.Flags().StringP(webRootFlag, webRootShortHand, webRootDefault, webRootDescription)
	_ = cmd.MarkFlagDirname(webRootFlag)
	cmd.Flags().String(urlPrefixFlag, urlPrefixDefault, "url prefix")
}

func PreRunE(cmd *cobra.Command, args []string) error {
	webRoot, err := cmd.Flags().GetString(webRootFlag)
	if err != nil {
		return nil
	}

	urlPrefix, err := cmd.Flags().GetString(urlPrefixFlag)
	if err != nil {
		return nil
	}

	files, err := loadFiles(webRoot)
	if err != nil {
		return nil
	}
	_ = files

	http.Handle("/", http.StripPrefix(urlPrefix, http.FileServer(http.Dir(webRoot))))
	return nil
}

type file struct {
	path string
	etag string

	filePath  string
	modTime   time.Time
	encodings smallestToLargest
}

func loadFiles(d string) ([]*file, error) {
	var ret []*file
	err := filepath.Walk(d, func(dp string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f := file{
			modTime:  info.ModTime(),
			filePath: dp,
		}

		fd, err := os.Open(dp)
		if err != nil {
			return err
		}
		defer fd.Close()

		id, err := f.identityEncode(fd)
		if err != nil {
			return err
		}

		err = f.gzipEncode(bytes.NewReader(id))
		if err != nil {
			return err
		}

		err = f.deflateEncode(bytes.NewReader(id))
		if err != nil {
			return err
		}

		sort.Sort(f.encodings)

		ret = append(ret, &f)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (f *file) identityEncode(r io.Reader) ([]byte, error) {
	d, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("handler: failed to identity encode file: %w", err)
	}

	f.encodings = append(f.encodings, &encodedFile{
		encoding: encIdentity,
		data:     d,
	})
	return d, nil
}

func (f *file) deflateEncode(r io.Reader) error {
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.BestCompression)

	_, err := io.Copy(w, r)
	if err != nil {
		return fmt.Errorf("handler: failed to deflate encode file: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("handler: failed to deflate encode file: %w", err)
	}

	f.encodings = append(f.encodings, &encodedFile{
		encoding: encDeflate,
		data:     buf.Bytes(),
	})
	return nil
}

func (f *file) gzipEncode(r io.Reader) error {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)

	_, err := io.Copy(w, r)
	if err != nil {
		return fmt.Errorf("handler: failed to gzip encode file: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("handler: failed to gzip encode file: %w", err)
	}

	f.encodings = append(f.encodings, &encodedFile{
		encoding: encGzip,
		data:     buf.Bytes(),
	})
	return nil
}

//go:generate stringer -type=encoding -linecomment
type encoding int

const (
	encIdentity encoding = 1 << iota // identity
	encGzip                          // gzip
	encDeflate                       // deflate
)

type encodedFile struct {
	encoding encoding
	data     []byte
}

type smallestToLargest []*encodedFile

func (e smallestToLargest) Len() int {
	return len(e)
}

func (e smallestToLargest) Less(i, j int) bool {
	return len(e[i].data) < len(e[j].data)
}

func (e smallestToLargest) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
