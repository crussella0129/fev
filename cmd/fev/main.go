// cmd/fev/main.go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "fev",
	Short: "Fev — local-first agentic CLI",
	Long:  "Fev is a personal assistant for navigating the digital space. Model-agnostic, single binary, local-first.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fev v" + version)
		fmt.Println("Interactive mode not yet implemented. Use 'fev version' to verify installation.")
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Fev version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("fev v%s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
