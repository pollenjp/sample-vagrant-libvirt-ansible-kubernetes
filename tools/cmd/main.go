package main

import (
	"fmt"
	"os"
)

func main() {
	rootCmd.AddCommand(subCmdCopy)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
