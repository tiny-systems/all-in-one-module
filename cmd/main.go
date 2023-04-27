package main

import (
  "github.com/rs/zerolog"
  "github.com/rs/zerolog/log"
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
  "github.com/tiny-systems/module/cli"
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

  cli.RegisterCommands(rootCmd)
  if err := rootCmd.Execute(); err != nil {
    log.Fatal().Err(err).Msg("command execute")
  }
}
