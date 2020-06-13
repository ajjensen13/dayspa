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
	"cloud.google.com/go/logging"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ajjensen13/gke"
	"os"
	"path/filepath"
	"sort"

	"github.com/ajjensen13/dayspa/internal/manifest"
)

type logEntry struct {
	WebRoot  string   `json:"web_root"`
	Index    string   `json:"index"`
	Assets   []string `json:"assets"`
	Checksum string   `json:"checksum"`
}

func Ngsw(webroot string, lg gke.Logger) (*manifest.Site, error) {
	entry := logEntry{WebRoot: webroot}
	defer func() { lg.Log(logging.Entry{Severity: logging.Info, Payload: entry}) }()

	f, err := os.Open(filepath.Join(webroot, "ngsw.json"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var m manifest.Manifest
	err = json.NewDecoder(f).Decode(&m)
	if err != nil {
		return nil, err
	}

	result := manifest.Site{
		Index: m.Index,
	}

	c := sha256.New()
	for _, a := range m.AssetGroups {
		for _, url := range a.Urls {
			lazy := a.InstallMode == manifest.Lazy
			asset, err := newEncodedAsset(webroot, url, lazy)
			if err != nil {
				return nil, nil
			}

			result.Assets = append(result.Assets, asset)
		}
	}

	sort.Sort(result.Assets)

	for _, asset := range result.Assets {
		c.Write([]byte(asset.Etag))
		entry.Assets = append(entry.Assets, fmt.Sprintf("%s@%s %s", asset.File, asset.Etag, asset.ContentType))
	}

	result.Checksum = base64.StdEncoding.EncodeToString(c.Sum(nil))

	entry.Index = result.Index
	entry.Checksum = result.Checksum

	return &result, nil
}
