package main

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "github.com/tiny-systems/main/components/array"
	_ "github.com/tiny-systems/main/components/common"
	_ "github.com/tiny-systems/main/components/db"
	_ "github.com/tiny-systems/main/components/email"
	_ "github.com/tiny-systems/main/components/google"
	_ "github.com/tiny-systems/main/components/http"
	_ "github.com/tiny-systems/main/components/network"
	_ "github.com/tiny-systems/main/components/slack"
	_ "github.com/tiny-systems/main/components/template"
	"github.com/tiny-systems/module/cli"
	"os"
	"os/signal"
	"syscall"
)

// RootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "tiny-system's main module",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func main() {
	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	viper.AutomaticEnv()
	if viper.GetBool("debug") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cli.RegisterCommands(rootCmd)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Printf("command execute error: %v\n", err)
	}
}
