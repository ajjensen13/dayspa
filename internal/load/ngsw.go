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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ajjensen13/gke"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ajjensen13/dayspa/internal/manifest"
)

type ngswManifest struct {
	ConfigVersion uint32           `json:"configVersion"`
	Timestamp     uint64           `json:"timestamp"`
	Index         string           `json:"index"`
	AssetGroups   []ngswAssetGroup `json:"assetGroups"`
}

type ngswAssetGroup struct {
	Name        string   `json:"name"`
	InstallMode string   `json:"installMode"`
	UpdateMode  string   `json:"updateMode"`
	Urls        []string `json:"urls"`
	Patterns    []string `json:"patterns"`
}

type logEntry struct {
	WebRoot         string          `json:"web_root"`
	ManifestDetails manifestDetails `json:"manifest_details"`
	SiteDetails     siteDetails     `json:"site_details"`
}

type manifestDetails struct {
	Path     string
	Manifest *ngswManifest
}

type siteDetails struct {
	Index    string
	Assets   []string
	Checksum string
}

// Loads an ngsw.json based webroot into a site manifest.
func Ngsw(webroot string, lg gke.Logger) (*manifest.Site, error) {
	entry := logEntry{WebRoot: webroot}
	defer func() { lg.Info(gke.NewMsgData("loaded ngsw.json", entry)) }()

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

func loadAssets(webroot string, assets []ngswAssetGroup) (manifest.EncodedAssets, error) {
	// First, load files from the manifest
	var result manifest.EncodedAssets
	for _, a := range assets {
		for _, url := range a.Urls {
			url = path.Clean(url) // use consistent cleaning with assets from manifest and from filesystem

			lazy := a.InstallMode == "lazy"
			asset, err := newEncodedAsset(webroot, url, lazy, "ngsw.json")
			if err != nil {
				return nil, fmt.Errorf("failed to build encoded asset from manifest %s: %w", url, err)
			}

			result = append(result, asset)
		}
	}

	// Next, load files not listed in the manifest
	err := filepath.Walk(webroot, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasPrefix(filepath.Base(fpath), ".") {
			return nil
		}

		if strings.HasPrefix(filepath.Base(fpath), "_") {
			return nil
		}

		rfp, err := filepath.Rel(webroot, fpath)
		if err != nil {
			return fmt.Errorf("failed to determine relative path to file %s from webroot %s: %w", fpath, webroot, err)
		}

		url := path.Join("/", filepath.ToSlash(rfp))

		if result.Contains(url) {
			return nil
		}

		asset, err := newEncodedAsset(webroot, url, true, "filesystem") // anything not in the manifest is assumed to be lazy-loaded
		if err != nil {
			return fmt.Errorf("failed to build encoded asset from file %s: %w", fpath, err)
		}

		result = append(result, asset)
		return nil
	})

	if err != nil {
		return nil, err
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
