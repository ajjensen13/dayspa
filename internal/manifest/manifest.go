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

type Mode string

const (
	Prefetch Mode = "prefetch"
	Lazy     Mode = "lazy"
)

type Manifest struct {
	ConfigVersion uint32       `json:"configVersion"`
	Timestamp     uint64       `json:"timestamp"`
	Index         string       `json:"index"`
	AssetGroups   []AssetGroup `json:"assetGroups"`
}

type AssetGroup struct {
	Name        string   `json:"name"`
	InstallMode Mode     `json:"installMode"`
	UpdateMode  Mode     `json:"updateMode"`
	Urls        []string `json:"urls"`
	Patterns    []string `json:"patterns"`
}

//go:generate stringer -type Encoding -linecomment
type Encoding int

const (
	Identity Encoding = iota // identity
	Gzip                     // gzip
	Deflate                  // deflate
)

type ContentType string

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

type EncodedAsset struct {
	Url         string
	File        string
	Lazy        bool
	ModTime     time.Time
	ContentType ContentType
	Etag        string
	Data        EncodedData
}

type EncodedDatum struct {
	Encoding Encoding
	Data     []byte
}

type EncodedData []*EncodedDatum

func (e EncodedData) Len() int {
	return len(e)
}

func (e EncodedData) Less(i, j int) bool {
	return len(e[i].Data) < len(e[j].Data)
}

func (e EncodedData) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

type Site struct {
	Index    string
	Checksum string
	Assets   EncodedAssets
}

type EncodedAssets []*EncodedAsset

func (e EncodedAssets) Len() int {
	return len(e)
}

func (e EncodedAssets) Less(i, j int) bool {
	return e[i].ContentType.Priority() < e[j].ContentType.Priority()
}

func (e EncodedAssets) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
