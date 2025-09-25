/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/ashupednekar/compose/pkg/creds"
	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate to remote registry",
	Long: `
	compose supports two main ways of authenticating to the registry
	Docker config
	  All you need to do is make sure your docker/podman is authenticated, we'll pick up the same creds
	Token refresher
	  This is a long running task you'd run in a tmux or as a linux service, pulls tokens and authenticates periodically
	`,
	Run: func(cmd *cobra.Command, args []string) {
		method, err := cmd.Flags().GetString("method")
		if err != nil{
			fmt.Printf("error reading method flag: %s", err)
			return
		}
		engine, err := cmd.Flags().GetString("engine")
		if err != nil {
			fmt.Printf("error reading engine flag: %s\n", err)
			return
		}
		if _, err := creds.AuthenticateWithRegistry(method, engine); err != nil{
			fmt.Printf("error authenticating to registry: %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:

	loginCmd.Flags().StringP("method", "m", "dockerconfig", "Select between dockerconfig or tokenrefresher")
	loginCmd.Flags().StringP("engine", "e", "docker", "Select between docker and podman")
	loginCmd.Flags().BoolP("force", "f", false, "Force login, ignoring existing credentials")
	// loginCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
