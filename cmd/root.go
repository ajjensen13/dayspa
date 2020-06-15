/*
Copyright © 2020 A. Jensen <jensen.aaro@gmail.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"context"
	"errors"
	"time"

	"fmt"
	"github.com/spf13/cobra"
	"net/http"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	"github.com/ajjensen13/gke"

	"github.com/ajjensen13/dayspa/internal/load"
	"github.com/ajjensen13/dayspa/internal/manifest"
	"github.com/ajjensen13/dayspa/internal/serve"
)

var rootCmd = &cobra.Command{
	Use: "dayspa",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancelFunc := gke.AliveContext()
		defer cancelFunc()

		logc, cleanup, err := gke.NewLogClient(ctx)
		if err != nil {
			panic(err)
		}
		defer cleanup()

		lg := logc.Logger("dayspa")

		m, ok := gke.Metadata()
		lg.Info("gke.Metadata()", m, ok)

		srv, err := InjectServer(ctx, lg, cmd)
		if err != nil {
			panic(lg.ErrorErr(err))
		}

		gke.Do(func(ctx context.Context) error {
			switch err := srv.ListenAndServe(); {
			case errors.Is(err, http.ErrServerClosed):
				lg.Noticef("server shutdown gracefully")
				return nil
			default:
				return lg.ErrorErr(err)
			}
		})

		<-gke.AfterAliveContext(time.Second * 10).Done()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var (
	cfgFile string
	mode    string
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.dayspa.yaml)")
	const modeFlag = "mode"
	rootCmd.PersistentFlags().StringVarP(&mode, modeFlag, "m", "", "mode to use (currently, only \"ngsw\" is supported)")
	const webrootFlag = "webroot"
	rootCmd.PersistentFlags().StringP(webrootFlag, "w", ".", "Web root directory")
	const addrFlag = "addr"
	rootCmd.PersistentFlags().StringP(addrFlag, "a", ":http", "address to listen on")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".dayspa" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".dayspa")
	}

	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

type webRoot string

func provideWebRoot(cmd *cobra.Command) (webRoot, error) {
	result, err := cmd.Flags().GetString("webroot")
	if err != nil {
		return "", err
	}
	return webRoot(result), nil
}

type modeType string

func provideMode(cmd *cobra.Command) (modeType, error) {
	result, err := cmd.Flags().GetString("mode")
	if err != nil {
		return "", err
	}
	return modeType(result), nil
}

func provideSite(webroot webRoot, mode modeType, lg gke.Logger) (*manifest.Site, error) {
	switch mode {
	case "ngsw":
		return load.Ngsw(string(webroot), lg)
	default:
		return nil, fmt.Errorf("unsupported mode: %s (try --mode=ngsw)", mode)
	}
}

func provideHandler(site *manifest.Site, lg gke.Logger) http.Handler {
	return serve.Handler(site, lg)
}

type addrType string

func provideAddr(cmd *cobra.Command) (addrType, error) {
	result, err := cmd.Flags().GetString("addr")
	if err != nil {
		return "", err
	}
	return addrType(result), nil
}

func provideServer(ctx context.Context, handler http.Handler, lg gke.Logger, addr addrType) (*http.Server, error) {
	result, err := gke.NewServer(ctx, handler, lg)
	if err != nil {
		return nil, err
	}
	result.Addr = string(addr)
	return result, nil
}
