package sub

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/pollenjp/sample-vagrant-libvirt-ansible-kubernete/tools/cmd/config"
	"github.com/spf13/cobra"
)

func NewCmdVagrantUp() *cobra.Command {
	return &cobra.Command{
		Use:   "vagrant-up",
		Short: "Vagrant Up",
		Run: func(_ *cobra.Command, _ []string) {
			if err := runVagrantUp(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}
}

func runVagrantUp() error {
	ctx := context.Background()

	if err := vagrantUp(ctx); err != nil {
		return err
	}

	return nil
}

func vagrantUp(ctx context.Context) error {
	cmdList := []*exec.Cmd{}

	// 各 host に対して vagrant up (一度に多くのVMを起動すると失敗する場合があるため、1つずつ起動する)
	for _, host := range config.VagrantHosts {
		cmd := exec.Command("vagrant", "up", host)
		cmdList = append(cmdList, cmd)
	}

	for _, cmd := range cmdList {
		if err := runCmdWithEachLineOutput(ctx, cmd); err != nil {
			return err
		}
	}

	return nil
}
