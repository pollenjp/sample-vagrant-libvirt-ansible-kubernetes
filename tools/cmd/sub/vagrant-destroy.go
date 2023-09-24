package sub

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func NewCmdVagrantDestroy() *cobra.Command {
	return &cobra.Command{
		Use:   "vagrant-rm",
		Short: "remove vagrant VMs",
		Long:  "'vagrant destroy'",
		Run: func(_ *cobra.Command, _ []string) {
			if err := executeVagrantDestroy(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}
}

func executeVagrantDestroy() error {
	ctx := context.Background()

	if err := runCmdWithEachLineOutput(ctx, exec.Command("vagrant", "destroy", "--force", "--graceful")); err != nil {
		return err
	}

	return nil
}
