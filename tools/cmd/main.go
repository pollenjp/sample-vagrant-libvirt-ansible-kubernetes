package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pollenjp/sample-vagrant-libvirt-ansible-kubernete/tools/cmd/sub"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use: "root",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Help()
		},
	}
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	rootCmd.AddCommand(sub.NewCmdCopy())
	rootCmd.AddCommand(sub.NewCmdSetupVagrantK8s())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}
