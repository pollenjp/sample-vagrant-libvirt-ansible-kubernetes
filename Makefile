INVENTORY_FILE :=
# vagrant, development, staging, production

SHELL := /bin/bash
PLAYBOOK_DIR := playbooks
PLAYBOOK :=

export

.DEFAULT_GOAL := help

.PHONY: install
install:  ## install requirements
	rye sync
	rye run ansible-galaxy collection install -r ansible-galaxy-requirements.yml

############
# Molecule #
############

.PHONY: molecule-test
molecule-test:
	${MAKE} molecule-template \
		MOLECULE_CMD="rye run molecule test"

.PHONY: molecule-destroy
molecule-destroy:
	${MAKE} molecule-template \
		MOLECULE_CMD="rye run molecule destroy"

.PHONY: molecule-template
molecule-template:  ## Run molecule tests for each role
ifndef MOLECULE_CMD
	${MAKE} error ERROR_MESSAGE="You must specify a 'MOLECULE_CMD' variable"
endif
	(\
		set -eux;\
		for role in $$(ls ${PLAYBOOK_DIR}/roles); do \
			echo "Running molecule tests for role: $$role"; \
			cd "${PLAYBOOK_DIR}/roles/$$role"; \
			if [[ -d "molecule" ]]; then \
				printf "\033[48;2;%d;%d;%dm\033[38;2;%d;%d;%dm" 0 0 238 255 255 255;\
				printf "Run molecule '${MOLECULE_CMD}' for $$role";\
				printf "\e[0m\n";\
				${MOLECULE_CMD} ;\
				rye run molecule test; \
			else \
				echo "No molecule tests found for role: $$role"; \
			fi; \
			cd -; \
		done;\
	)

.PHONY: lint
lint:  ## run yamllint
	rye run yamllint -c ".yamllint" --strict \
		"${PLAYBOOK_DIR}" \
		"inventory"
	rye run ansible-lint "${PLAYBOOK_DIR}"

.PHONY: run
run:  ## run the playbook. Need INVENTORY_FILE
ifndef INVENTORY_FILE
	${MAKE} error ERROR_MESSAGE="You must specify a 'INVENTORY_FILE' variable"
endif
	[[ -f "${INVENTORY_FILE}" ]]
	[[ -f "${PLAYBOOK}" ]]
	rye run ansible-playbook \
		-vvv \
		-i "${INVENTORY_FILE}" \
		"${PLAYBOOK}"

.PHONY: debug-k8s-setup
debug-k8s-setup:  ## debug the playbook (vagrant)
#	${MAKE} clean
	${MAKE} vagrant-up
	-vagrant ssh-config > inventory/vagrant.ssh_config
	ANSIBLE_SSH_ARGS='-F inventory/vagrant.ssh_config' \
		${MAKE} run \
			INVENTORY_FILE=inventory/vagrant.py \
			PLAYBOOK=playbooks/dns_server.yml
	ANSIBLE_SSH_ARGS='-F inventory/vagrant.ssh_config' \
		${MAKE} run \
			INVENTORY_FILE=inventory/vagrant.py \
			PLAYBOOK=playbooks/k8s-setup-control-plane.yml
	ANSIBLE_SSH_ARGS='-F inventory/vagrant.ssh_config' \
		${MAKE} run \
			INVENTORY_FILE=inventory/vagrant.py \
			PLAYBOOK="playbooks/k8s-setup-join-node.yml"

.PHONY: debug
debug:
#	${MAKE} vagrant-up
	-vagrant ssh-config > inventory/vagrant.ssh_config
	ANSIBLE_SSH_ARGS='-F inventory/vagrant.ssh_config' \
		rye run ansible-playbook \
			-i inventory/vagrant.py \
			playbooks/debug.yml

.PHONY: clean
clean:  ## halt and destroy vagrant
	-${MAKE} vagrant-halt
	-${MAKE} vagrant-destroy

vagrant-up:  ##
	vagrant box update
	vagrant up vm-dns.vagrant.home
	vagrant up vm01.vagrant.home
	vagrant up vm02.vagrant.home
	vagrant up vm03.vagrant.home
	vagrant up vm04.vagrant.home

vagrant-halt:  ##
	vagrant halt

vagrant-destroy:  ##
	-${MAKE} vagrant-halt
	vagrant destroy -f

#########
# Utils #
#########

.PHONY : error
error:  ## utils for error message
	@printf "\033[48;2;%d;%d;%dm" 255  85  85
	@printf "\033[38;2;%d;%d;%dm" 255 255 255
	@printf "%s" "${ERROR_MESSAGE}"
	@printf "\e[0m\n"
	@${MAKE} interupt_make

.PHONY : interupt_make
interupt_make:  ## interrupt make command
	$(error "${ERROR_MESSAGE}")

.PHONY : help
help:  ## show help
	@cat $(MAKEFILE_LIST) \
		| grep -E '^[.a-zA-Z0-9_-]+ *:.*##.*' \
		| xargs -I'<>' \
			bash -c "\
				printf '<>' | awk -F'[:]' '{ printf \"\033[36m%-15s\033[0m\", \$$1 }'; \
				printf '<>' | awk -F'[##]' '{ printf \"%s\n\", \$$3 }'; \
			"
