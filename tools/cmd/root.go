package main

import (
	"github.com/pollenjp/sample-vagrant-libvirt-ansible-kubernetes/tools/cmd/sub"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use: "root",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Help()
		},
	}
	subCmdCopy = &cobra.Command{
		Use:   "copy",
		Short: "copy files",
		Long:  `copy files`,
		Run: func(_ *cobra.Command, _ []string) {
			sub.Copy()
		},
	}
)
