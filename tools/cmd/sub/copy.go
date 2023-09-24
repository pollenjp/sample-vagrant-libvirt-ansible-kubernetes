package sub

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

func NewCmdCopy() *cobra.Command {
	return &cobra.Command{
		Use:   "copy",
		Short: "copy files",
		Run: func(_ *cobra.Command, _ []string) {
			if err := copy(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}
}

type SrcDest struct {
	src  string
	dest string
}

func copy() error {
	ctx := context.Background()

	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	srcRoot := filepath.Join(home, "workdir/github.com/pollenjp/infra/ansible")
	destRoot := filepath.Join(home, "workdir/github.com/pollenjp/sample-vagrant-libvirt-ansible-kubernetes")

	if err := runCmdWithEachLineOutput(ctx, exec.Command("rm", "-rfv", filepath.Join(destRoot, "playbooks"))); err != nil {
		return err
	}

	if err := runCmdWithEachLineOutput(ctx, exec.Command("rm", "-rfv", filepath.Join(destRoot, "tools", "cmd"))); err != nil {
		return err
	}

	relativePaths := []string{
		".gitignore",
		".yamllint",
		"ansible.cfg",
		"Gemfile",
		"Gemfile.lock",
		"Makefile",
		"pyproject.toml",
		"Vagrantfile",
		"ansible-galaxy-requirements.yml",
		"inventory/.gitignore",
		"inventory/vagrant.py",
		"playbooks/.gitignore",
		"playbooks/dns_server.yml",
		"playbooks/k8s-setup-control-plane.yml",
		"playbooks/k8s-setup-join-node.yml",
		"playbooks/config/kube-flannel.yml",
		"playbooks/files/playbooks/dns_server",
		"playbooks/group_vars/k8s_all",
		"playbooks/roles/dns_server",
		"playbooks/roles/install_bind",
		"playbooks/roles/install_docker",
		"playbooks/roles/install_kubernetes",
		"playbooks/roles/k8s_cp_kubeadm_init",
		"playbooks/roles/k8s_cp_load_balancer",
		"playbooks/roles/k8s_requirements",
		"playbooks/roles/utils",
		"tools/cmd",
	}

	copyTargets := []SrcDest{}
	for _, relativePath := range relativePaths {
		src := filepath.Join(srcRoot, relativePath)
		dest := filepath.Join(destRoot, relativePath)
		copyTargets = append(copyTargets, SrcDest{src: src, dest: dest})
	}

	for _, target := range copyTargets {
		fmt.Printf("%s -> %s\n", target.src, target.dest)
		fileInfo, err := os.Stat(target.src)
		if err != nil {
			return fmt.Errorf("file info: %w", err)
		}

		// create parent directory
		if err := os.MkdirAll(filepath.Dir(target.dest), 0755); err != nil {
			return fmt.Errorf("creating: %w", err)
		}

		// If target.src is directory add slash to the end of target.src
		// rsync requires slash at the end of directory, otherwise it will create directory with the same name under dest path.
		src := target.src
		if fileInfo.IsDir() {
			src += "/"
		}

		if err := runCmdWithEachLineOutput(ctx, exec.Command("rsync", "-a", src, target.dest)); err != nil {
			return err
		}
	}

	return nil
}
