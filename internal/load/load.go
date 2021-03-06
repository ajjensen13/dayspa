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
	"github.com/ajjensen13/gke"

	"github.com/ajjensen13/dayspa/internal/load/filesystem"
	"github.com/ajjensen13/dayspa/internal/load/ngsw"
	"github.com/ajjensen13/dayspa/internal/manifest"
)

// Ngsw loads an ngsw.json based webroot into a site manifest.
func Ngsw(webroot string, lg gke.Logger) (*manifest.Site, error) {
	return ngsw.Load(webroot, lg)
}

// Filesystem loads a filesystem based webroot into a site manifest.
func Filesystem(webroot string, lg gke.Logger) (*manifest.Site, error) {
	return filesystem.Load(webroot, lg)
}
