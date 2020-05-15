package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"

	"github.com/ajjensen13/dayspa/internal/handler"
	"github.com/ajjensen13/dayspa/internal/server"
)

func init() {
	cobra.OnInitialize(initConfig)
	server.Init(&rootCmd)
	handler.Init(&rootCmd)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = cobra.Command{
	Use:   "dayspa",
	Short: "An Single-Page Web App Server",
	Long:  `A Single-Page App Web Server`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return handler.PreRunE(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.RunE(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
