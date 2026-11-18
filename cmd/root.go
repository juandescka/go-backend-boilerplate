package cmd

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "hris-core-service",
	Short: "HRIS core backend service CLI",
	Long:  "CLI for managing the HRIS core backend service",
}

func Execute() error {
	return RootCmd.Execute()
}
