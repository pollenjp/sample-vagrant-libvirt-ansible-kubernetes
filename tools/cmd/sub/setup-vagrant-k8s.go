package sub

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	sshConfig = "inventory/vagrant.ssh_config"
	inventory = "inventory/vagrant.py"
)

func NewCmdSetupVagrantK8s() *cobra.Command {
	return &cobra.Command{
		Use:   "setup-vagrant-k8s",
		Short: "setup k8s with vagrant",
		Run: func(_ *cobra.Command, _ []string) {
			if err := setupVagrantK8s(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}
}

func setupVagrantK8s() error {
	ctx := context.Background()

	// if err := runCmdWithEachLineOutput(ctx, exec.Command("vagrant", "destroy", "-fg")); err != nil {
	// 	return err
	// }

	if err := vagrantUp(ctx); err != nil {
		return err
	}

	if err := runCmdWithEachLineOutput(ctx, exec.Command(
		"bash",
		"-c",
		fmt.Sprintf("vagrant ssh-config > %s", sshConfig),
	)); err != nil {
		return err
	}

	if err := runAnsiblePlaybook(
		ctx,
		[]string{
			"playbooks/dns_server.yml",
			"playbooks/k8s-setup-control-plane.yml",
			"playbooks/k8s-setup-join-node.yml",
		},
	); err != nil {
		return err
	}

	return nil
}

func runAnsiblePlaybook(ctx context.Context, playbooks []string) error {
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
		if err := runCmdWithEachLineOutput(ctx, cmd); err != nil {
			return err
		}
	}

	return nil
}

func runCmdWithEachLineOutput(ctx context.Context, cmd *exec.Cmd) error {
	fmt.Printf("run command: %s\n", cmd.Args)

	reader, writer := io.Pipe()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // to kill process group
	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup

	wg.Add(1)
	// get command output and manipulate it
	go func() {
		defer wg.Done()

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() { // false after writer.Close() and all data are read
			log.Printf("[%s:%d] %s\n", cmd.Args[0], cmd.Process.Pid, scanner.Text())
		}
	}()

	var cmdExitErr error
	// command exit context
	cmdCtx, cmdDone := context.WithCancel(ctx)

	wg.Add(1)
	// command の終了を待つ goroutine
	go func() {
		cmdExitErr = cmd.Wait()
		cmdDone()
		writer.Close()
		wg.Done()
	}()

	interruptSig := make(chan os.Signal, 1)
	signal.Notify(interruptSig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-cmdCtx.Done(): // exit success
	case sig := <-interruptSig:
		log.Printf("try to send signal (%s) to PID (%d)\n", sig, cmd.Process.Pid)

		// WARNING: (*os.Process).Signal Sending Interrupt on Windows is not implemented.
		if err := cmd.Process.Signal(sig); err != nil {
			// log.Printf("failed to send signal %s: %s\n", s, err)
			return fmt.Errorf("failed to send signal %s: %w", sig, err)
		}

		timeoutDur := 20 * time.Second
		fmt.Printf("wait %s second for command exit\n", timeoutDur)

		select {
		case <-cmdCtx.Done(): // successfully interrupted
			log.Printf("successfully interrupted PID (%d)\n", cmd.Process.Pid)
		case <-time.After(timeoutDur): // timeout
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			fmt.Printf("force kill PID (%d)\n", cmd.Process.Pid)
			<-cmdCtx.Done()
		}
	}

	wg.Wait()

	if cmdExitErr != nil {
		return fmt.Errorf("%w %s", cmdExitErr, cmd.Args)
	}
	return nil
}
