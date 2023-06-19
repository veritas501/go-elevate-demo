package cmd

import (
	"github.com/spf13/cobra"
	"github.com/veritas501/go-elevate-demo/pkg/elevate"
	"log"
	"os"
	"os/exec"
)

func init() {
	// disable completion options
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// add elevate cmdline to cobra
	elevate.AddCmdlineToCobra(rootCmd)
}

func entryPoint(cmd *cobra.Command, args []string) {
	// this function will run with ADMIN
	a := exec.Command("cmd", "/c", "chcp 65001 && whoami /priv")
	a.Stdout = os.Stdout
	a.Run()
}

var rootCmd = &cobra.Command{
	Use: "go-elevate-demo",
	Run: func(cmd *cobra.Command, args []string) {
		elevate.Run(cmd, args, entryPoint)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
