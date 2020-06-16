/*
 * Copyright © 2020  A. Jensen <jensen.aaro@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package load

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ajjensen13/dayspa/internal/manifest"
)

func identityEncoded(fpath string) (*manifest.EncodedDatum, error) {
	data, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	return &manifest.EncodedDatum{ContentEncoding: manifest.Identity, Data: data}, nil
}

func gzipEncoded(raw []byte) (*manifest.EncodedDatum, error) {
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	_, err = gz.Write(raw)
	if err != nil {
		return nil, err
	}

	err = gz.Close()
	if err != nil {
		return nil, err
	}

	return &manifest.EncodedDatum{ContentEncoding: manifest.Gzip, Data: buf.Bytes()}, nil
}

func flateEncoded(raw []byte) (*manifest.EncodedDatum, error) {
	var buf bytes.Buffer
	fl, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		return nil, err
	}

	_, err = fl.Write(raw)
	if err != nil {
		return nil, err
	}

	err = fl.Close()
	if err != nil {
		return nil, err
	}

	return &manifest.EncodedDatum{ContentEncoding: manifest.Deflate, Data: buf.Bytes()}, nil
}

func calculateETag(raw []byte) string {
	hash := sha256.Sum256(raw)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func newEncodedAsset(webroot, url string, lazy bool, source string) (*manifest.EncodedAsset, error) {
	fpath := filepath.FromSlash(url)
	fpath = filepath.Join(webroot, url)

	fi, err := os.Stat(fpath)
	if err != nil {
		return nil, err
	}

	result := manifest.EncodedAsset{
		Url:     url,
		File:    fpath,
		Lazy:    lazy,
		Source:  source,
		ModTime: fi.ModTime(),
	}

	raw, err := identityEncoded(fpath)
	if err != nil {
		return nil, err
	}
	result.Data = append(result.Data, raw)

	gz, err := gzipEncoded(raw.Data)
	if err != nil {
		return nil, err
	}
	result.Data = append(result.Data, gz)

	fl, err := flateEncoded(raw.Data)
	if err != nil {
		return nil, err
	}
	result.Data = append(result.Data, fl)

	result.ContentType = determineContentType(fpath, raw.Data)
	result.Etag = calculateETag(raw.Data)

	sort.Sort(result.Data)

	return &result, nil
}

func determineContentType(fpath string, data []byte) manifest.ContentType {
	result := ""

	if ext := filepath.Ext(fpath); ext != "" {
		result = mime.TypeByExtension(ext)
	}

	if result == "" {
		result = http.DetectContentType(data)
	}

	result = strings.ToLower(result)

	switch result {
	case "application/javascript", "application/ecmascript", "application/x-ecmascript", "application/x-javascript", "text/javascript", "text/ecmascript", "text/javascript1.0", "text/javascript1.1", "text/javascript1.2", "text/javascript1.3", "text/javascript1.4", "text/javascript1.5", "text/jscript", "text/livescript", "text/x-ecmascript", "text/x-javascript":
		result = "text/javascript"
	case "":
		result = "application/octet-stream"
	}

	return manifest.ContentType(result)
}
