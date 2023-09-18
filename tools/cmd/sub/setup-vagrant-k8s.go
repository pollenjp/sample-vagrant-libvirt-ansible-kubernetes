package sub

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	sshConfig    = "inventory/vagrant.ssh_config"
	inventory    = "inventory/vagrant.py"
	vagrantHosts = []string{
		"vm-dns.vagrant.home",
		"vm01.vagrant.home",
		"vm02.vagrant.home",
	}
)

func NewCmdSetupVagrantK8s() *cobra.Command {
	return &cobra.Command{
		Use:   "setup-vagrant-k8s",
		Short: "setup k8s with vagrant",
		Run: func(_ *cobra.Command, _ []string) {
			if err := setupVagrantK8s(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
}

func setupVagrantK8s() error {
	if err := vagrantUp(); err != nil {
		return err
	}

	if err := runCmdWithEachLineOutput(exec.Command(
		"bash",
		"-c",
		fmt.Sprintf("vagrant ssh-config > %s", sshConfig),
	)); err != nil {
		return err
	}

	if err := runAnsiblePlaybook(); err != nil {
		return err
	}

	return nil
}

func runAnsiblePlaybook() error {
	playbooks := []string{
		// "playbooks/dns_server.yml",
		// "playbooks/k8s-setup-control-plane.yml",
		"playbooks/k8s-setup-join-node.yml",
	}
	cmdList := []*exec.Cmd{}
	for _, playbook := range playbooks {
		playbook, err := filepath.Abs(playbook)
		if err != nil {
			return err
		}
		cmd := exec.Command("rye", "run", "ansible-playbook", "-i", inventory, playbook)
		cmd.Env = append(
			cmd.Environ(),
			fmt.Sprintf("ANSIBLE_SSH_ARGS=-F %s", sshConfig),
		)
		cmdList = append(cmdList, cmd)
	}

	for _, cmd := range cmdList {
		if err := runCmdWithEachLineOutput(cmd); err != nil {
			return err
		}
	}

	return nil
}

func vagrantUp() error {
	cmdList := []*exec.Cmd{}

	// 各 host に対して vagrant up (一度に多くのVMを起動すると失敗する場合があるため、1つずつ起動する)
	for _, host := range vagrantHosts {
		cmd := exec.Command("vagrant", "up", host)
		cmdList = append(cmdList, cmd)
	}

	for _, cmd := range cmdList {
		if err := runCmdWithEachLineOutput(cmd); err != nil {
			return fmt.Errorf("command (%s %s): %w", cmd.Path, cmd.Args, err)
		}
	}

	return nil
}

func runCmdWithEachLineOutput(cmd *exec.Cmd) error {
	reader, writer := io.Pipe()

	cmdCtx, cmdCtxCancel := context.WithCancel(context.Background())

	scannerStopped := make(chan struct{})
	go func() {
		defer close(scannerStopped)

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			log.Printf("[%s] %s\n", cmd.Args[0], scanner.Text())
		}
	}()

	var cmdExitErr error

	cmd.Stdout = writer
	_ = cmd.Start()
	go func() {
		cmdExitErr = cmd.Wait()
		cmdCtxCancel()
		writer.Close()
	}()
	<-cmdCtx.Done()
	<-scannerStopped

	if cmdExitErr != nil {
		return fmt.Errorf("%w %s", cmdExitErr, cmd.Args)
	}
	return nil
}
