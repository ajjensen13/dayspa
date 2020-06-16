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

// Package serve provides an http.Handler for dayspa apps.
package serve

import (
	"errors"
	"fmt"
	"github.com/ajjensen13/gke"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ajjensen13/dayspa/internal/manifest"
)

// Handler returns an http.Handler that serves a manifest.
func Handler(site *manifest.Site, lg gke.Logger) http.Handler {
	result := handler{
		Index:      site.Index,
		Assets:     site.Assets,
		Checksum:   site.Checksum,
		LookupPath: make(map[string]*manifest.EncodedAsset, len(site.Assets)),
		Logger:     lg,
	}

	for _, asset := range site.Assets {
		result.LookupPath[asset.Url] = asset
	}

	for url, asset := range result.LookupPath {
		if path.Base(url) != "index.html" {
			continue
		}

		dir := path.Dir(url)
		if result.LookupPath[dir] != nil {
			continue
		}

		result.LookupPath[dir] = asset
	}

	return &result
}

type handler struct {
	Index      string
	LookupPath map[string]*manifest.EncodedAsset
	Assets     manifest.EncodedAssets
	Checksum   string
	Logger     gke.Logger
}

type logEntry struct {
	RequestDetails requestDetails `json:"request_details"`
	ServeDetails   serveDetails   `json:"serve_details"`
	PushDetails    pushDetails    `json:"push_details"`
}

type requestDetails struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Host   string `json:"host"`
}

func (r requestDetails) String() string {
	return fmt.Sprintf("%s %s%s", r.Method, r.Host, r.Path)
}

type serveDetails struct {
	Status int `json:"status"`
	Size   int `json:"size"`
}

var (
	boolTrue  = true
	boolFalse = false
)

type pushDetails struct {
	RequestTriggersPush *bool    `json:"request_triggers_push"`
	ClientSupportsPush  *bool    `json:"client_supports_push"`
	ClientNeedsAssets   *bool    `json:"client_needs_assets"`
	PushAttempted       bool     `json:"push_attempted"`
	ServerChecksum      string   `json:"server_checksum,omitempty"`
	ClientChecksum      string   `json:"client_checksum,omitempty"`
	Assets              []string `json:"assets"`
}

const pushCookieName = "_dayspa_push"

func (h *handler) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	entry := logEntry{RequestDetails: requestDetails{
		Method: r.Method,
		Host:   r.Host,
		Path:   r.URL.Path,
	}}
	defer func() { h.Logger.Info(gke.NewMsgData(entry.RequestDetails.String(), entry)) }()

	entry.PushDetails = h.tryPush(wr, r)
	entry.ServeDetails = h.serveAsset(wr, r)
}

func requestTriggersPush(p string, index string) bool {
	return p == index || filepath.Ext(p) == ""
}

func (h *handler) tryPush(wr http.ResponseWriter, r *http.Request) (result pushDetails) {
	result.ServerChecksum = h.Checksum
	if !requestTriggersPush(r.URL.Path, h.Index) {
		result.RequestTriggersPush = &boolFalse
		return
	}
	result.RequestTriggersPush = &boolTrue

	pusher, ok := wr.(http.Pusher)
	if !ok {
		result.ClientSupportsPush = &boolFalse
		return
	}
	result.ClientSupportsPush = &boolTrue

	if clientHasAssets(r, h.Checksum, &result) {
		result.ClientNeedsAssets = &boolFalse
		return
	}
	result.ClientNeedsAssets = &boolTrue

	http.SetCookie(wr, &http.Cookie{
		Name:     pushCookieName,
		Value:    h.Checksum,
		Path:     "/",
		Domain:   r.Host,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(time.Hour * 24 * 365 / time.Second), // 1 year in seconds
	})

	result.PushAttempted = true
	result.Assets = make([]string, 0, len(h.Assets))

	opts := http.PushOptions{Method: http.MethodGet}
	for _, asset := range h.Assets {
		if asset.Lazy {
			continue
		}

		if requestTriggersPush(asset.Url, h.Index) {
			continue
		}

		result.Assets = append(result.Assets, asset.Url)
		_ = pusher.Push(asset.Url, &opts)
	}

	return
}

func clientHasAssets(r *http.Request, checksum string, p *pushDetails) bool {
	c, err := r.Cookie(pushCookieName)
	switch {
	case errors.Is(err, http.ErrNoCookie):
		return false
	case c.Value == checksum:
		p.ClientChecksum = c.Value
		return true
	default:
		p.ClientChecksum = c.Value
		return false
	}
}

func (h *handler) serveAsset(wr http.ResponseWriter, r *http.Request) (result serveDetails) {
	asset, ok := h.LookupPath[r.URL.Path]
	if !ok {
		result.Status = http.StatusNotFound
		http.NotFound(wr, r)
		return
	}

	if asset.Etag != "" {
		if etag := r.Header.Get("If-None-Match"); etag == asset.Etag {
			result.Status = http.StatusNotModified
			wr.WriteHeader(http.StatusNotModified)
			return
		}
	}

	header := wr.Header()
	header.Set("ETag", asset.Etag)
	header.Set("Content-Type", string(asset.ContentType))

	encodings := r.Header.Get("Accept-Encoding")
	for _, datum := range asset.Data {

		es := datum.ContentEncoding.String()
		if strings.Contains(encodings, es) || manifest.Identity == datum.ContentEncoding {
			header.Set("Content-Encoding", es)
		}

		result.Status = http.StatusOK
		wr.WriteHeader(http.StatusOK)

		_, err := wr.Write(datum.Data)
		if err != nil {
			panic(err)
		}

		result.Size = len(datum.Data)

		return
	}

	panic("no acceptable encoding found")
}
