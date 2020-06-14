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
	WebRoot         string          `json:"web_root"`
	ManifestDetails manifestDetails `json:"manifest_details"`
	SiteDetails     siteDetails     `json:"site_details"`
}

type manifestDetails struct {
	Path     string
	Manifest *manifest.Manifest
}

type siteDetails struct {
	Index    string
	Assets   []string
	Checksum string
}

func Ngsw(webroot string, lg gke.Logger) (*manifest.Site, error) {
	entry := logEntry{WebRoot: webroot}
	defer func() { lg.Log(logging.Entry{Severity: logging.Info, Payload: entry}) }()

	var err error
	entry.ManifestDetails, err = parseManifest(webroot)
	if err != nil {
		return nil, err
	}

	m := entry.ManifestDetails.Manifest

	result := manifest.Site{Index: m.Index}

	result.Assets, err = loadAssets(webroot, m.AssetGroups)
	if err != nil {
		return nil, err
	}

	c := sha256.New()
	for _, asset := range result.Assets {
		c.Write([]byte(asset.Etag))
		entry.SiteDetails.Assets = append(entry.SiteDetails.Assets, fmt.Sprintf("%s@%s %s", asset.File, asset.Etag, asset.ContentType))
	}

	result.Checksum = base64.StdEncoding.EncodeToString(c.Sum(nil))

	entry.SiteDetails.Index = result.Index
	entry.SiteDetails.Checksum = result.Checksum

	return &result, nil
}

func loadAssets(webroot string, assets []manifest.AssetGroup) (manifest.EncodedAssets, error) {
	var result manifest.EncodedAssets
	for _, a := range assets {
		for _, url := range a.Urls {
			lazy := a.InstallMode == manifest.Lazy
			asset, err := newEncodedAsset(webroot, url, lazy)
			if err != nil {
				return nil, err
			}

			result = append(result, asset)
		}
	}

	sort.Sort(result)
	return result, nil
}

func parseManifest(webroot string) (result manifestDetails, err error) {
	result.Path = filepath.Join(webroot, "ngsw.json")

	var f *os.File
	f, err = os.Open(result.Path)
	if err != nil {
		return
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&result.Manifest)
	return
}
