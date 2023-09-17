package sub

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type SrcDest struct {
	src  string
	dest string
}

func Copy() error {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	srcRoot := filepath.Join(dirname, "workdir/github/pollenjp/infra/ansible")
	destRoot := filepath.Join(dirname, "workdir/github/pollenjp/sample-vagrant-libvirt-ansible-kubernetes")

	relativePaths := []string{
		".yamllint",
		"Makefile",
		"ansible.cfg",
		"pyproject.toml",
		"Vagrantfile",
		"Gemfile",
		"Gemfile.lock",
		"ansible-galaxy-requirements.yml",
		"playbooks/.gitignore",
		"inventory/vagrant.py",
		"playbooks/dns_server.yml",
		"playbooks/k8s-setup-join-node.yml",
		"playbooks/config/kube-flannel.yml",
		"playbooks/roles/dns-server",
		"playbooks/roles/install-kubernetes",
		"playbooks/roles/install-docker",
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
