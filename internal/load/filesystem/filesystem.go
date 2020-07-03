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

package filesystem

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/ajjensen13/gke"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ajjensen13/dayspa/internal/load/log"
	"github.com/ajjensen13/dayspa/internal/load/shared"
	"github.com/ajjensen13/dayspa/internal/manifest"
)

// Loads filesystem based webroot into a site manifest.
func Load(webroot string, lg gke.Logger) (*manifest.Site, error) {
	entry := log.Entry{WebRoot: webroot}
	defer func() { lg.Info(gke.NewMsgData("loaded filesystem", entry)) }()

	result := manifest.Site{Index: "/index.html"}

	var err error
	result.Assets, err = loadAssets(webroot)
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

func loadAssets(webroot string) (manifest.EncodedAssets, error) {
	var result manifest.EncodedAssets
	err := filepath.Walk(webroot, func(fpath string, info os.FileInfo, err error) error {
		switch {
		case err != nil:
			return err
		case info.IsDir():
			return nil
		case strings.HasPrefix(filepath.Base(fpath), "."):
			return nil
		case strings.HasPrefix(filepath.Base(fpath), "_"):
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

		asset, err := shared.EncodedAsset(webroot, url, true, "filesystem")
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
