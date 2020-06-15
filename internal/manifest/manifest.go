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

package manifest

import (
	"strings"
	"time"
)

//go:generate stringer -type ContentEncoding -linecomment
type ContentEncoding int

const (
	// Identity Content-ContentEncoding (this is the default)
	Identity ContentEncoding = iota // identity
	// Gzip Content-ContentEncoding
	Gzip // gzip
	// Deflate Content-ContentEncoding
	Deflate // deflate
)

// ContentType is used to prioritize asset types based on the Critical Rendering Path.
// See: https://developers.google.com/web/fundamentals/performance/critical-rendering-path
type ContentType string

// Priority returns the ContentType's sort priority based on the Critical Rendering Path.
// See: https://developers.google.com/web/fundamentals/performance/critical-rendering-path
func (c ContentType) Priority() int {
	s := string(c)
	switch {
	case strings.HasPrefix(s, "text/html"):
		return 0
	case strings.HasPrefix(s, "text/css"):
		return 1
	case strings.HasPrefix(s, "text/javascript"):
		return 2
	default:
		return 3
	}
}

// EncodedAsset represents a single asset that has been loaded, and encoded.
type EncodedAsset struct {
	Url         string      `json:"url"`
	File        string      `json:"file"`
	Lazy        bool        `json:"lazy"`
	ModTime     time.Time   `json:"mod_time"`
	ContentType ContentType `json:"content_type"`
	Etag        string      `json:"etag"`
	Data        EncodedData `json:"-"`
}

// EncodedDatum represents a single encoding of a single asset.
type EncodedDatum struct {
	ContentEncoding ContentEncoding `json:"content_encoding"`
	Data            []byte          `json:"-"`
}

// EncodedData is a sorted list of EncodedDatum.
// The entries are sorted based on their ContentLength such that smaller
// entries occur before larger ones.
type EncodedData []*EncodedDatum

// Len implements sort.Interface.Len()
func (e EncodedData) Len() int {
	return len(e)
}

// Len implements sort.Interface.Less()
func (e EncodedData) Less(i, j int) bool {
	return len(e[i].Data) < len(e[j].Data)
}

// Len implements sort.Interface.Swap()
func (e EncodedData) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

// Site represents a loaded site.
type Site struct {
	Index    string        `json:"index"`
	Checksum string        `json:"checksum"`
	Assets   EncodedAssets `json:"assets"`
}

// EncodedAssets is a sorted list of EncodedAssets.
// The entries are sorted based on their ContentType.Priority() such that higher
// priority entries occur before lower priority entries.
type EncodedAssets []*EncodedAsset

// Len implements sort.Interface.Len()
func (e EncodedAssets) Len() int {
	return len(e)
}

// Len implements sort.Interface.Less()
func (e EncodedAssets) Less(i, j int) bool {
	return e[i].ContentType.Priority() < e[j].ContentType.Priority()
}

// Len implements sort.Interface.Swap()
func (e EncodedAssets) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
