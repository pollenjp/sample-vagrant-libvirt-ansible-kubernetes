package sub

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewCmdCopy() *cobra.Command {
	return &cobra.Command{
		Use:   "copy",
		Short: "copy files",
		Run: func(_ *cobra.Command, _ []string) {
			copy()
		},
	}
}

type SrcDest struct {
	src  string
	dest string
}

func copy() error {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	srcRoot := filepath.Join(dirname, "workdir/github/pollenjp/infra/ansible")
	destRoot := filepath.Join(dirname, "workdir/github/pollenjp/sample-vagrant-libvirt-ansible-kubernetes")

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
		"playbooks/roles/dns-server",
		"playbooks/roles/install-bind",
		"playbooks/roles/install-docker",
		"playbooks/roles/install-kubernetes",
		"playbooks/roles/k8s-control-plane",
		"playbooks/roles/k8s-requirements",
		"playbooks/roles/utils",
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

		// copy file
		if err := exec.Command("rsync", "-a", src, target.dest).Run(); err != nil {
			return fmt.Errorf("copying: %w", err)
		}
	}

	return nil
}
