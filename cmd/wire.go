// +build wireinject

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
package cmd

import (
	"context"
	"github.com/ajjensen13/gke"
	"github.com/google/wire"
	"github.com/spf13/cobra"
	"net/http"
)

func InjectServer(ctx context.Context, lg gke.Logger, cmd *cobra.Command) (*http.Server, error) {
	panic(wire.Build(provideWebRoot, provideSite, provideHandler, provideServer, provideMode, provideAddr))
}
