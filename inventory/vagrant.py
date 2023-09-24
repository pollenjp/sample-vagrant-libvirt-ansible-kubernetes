#!/usr/bin/env python3
# Standard Library
import io
import json
import logging
import subprocess
import sys
import typing as t
from logging import NullHandler
from logging import getLogger

# Third Party Library
import paramiko
from pydantic import BaseModel
from pydantic import ConfigDict
from pydantic import Field
from tap import Tap

logger = getLogger(__name__)
logger.addHandler(NullHandler())

Hostname = str
vagrant_domain = "vagrant.home"


class VagrantDomains(BaseModel):
    # VMs

    vm_dns: str = Field(default=f"vm-dns.{vagrant_domain}")
    vm01: str = Field(default=f"vm01.{vagrant_domain}")
    vm02: str = Field(default=f"vm02.{vagrant_domain}")
    vm03: str = Field(default=f"vm03.{vagrant_domain}")
    vm04: str = Field(default=f"vm04.{vagrant_domain}")

    # DNS

    ns1: str = Field(default=f"ns1.{vagrant_domain}")
    k8s_cp_endpoint: str = Field(default=f"k8s-cp-endpoint.{vagrant_domain}")

    # pydantic config

    model_config = ConfigDict(frozen=True)


vagrant_domains = VagrantDomains()


class Args(Tap):
    list: bool = False
    host: str | None = None
    dry_run: bool = True  # False のとき全てのホストが起動しているかチェックする


class VagrantProvisioningInfo(BaseModel):
    ipv4_address: str


class GroupModel(BaseModel):
    vars: dict[str, t.Any] | None = None
    hosts: list[Hostname] = Field(default_factory=list)
    children: list[str] = Field(default_factory=list)


class HostVars(BaseModel):
    vm_dns: dict[str, t.Any] = Field(serialization_alias=f"{vagrant_domains.vm_dns}")
    vm01: dict[str, t.Any] = Field(serialization_alias=f"{vagrant_domains.vm01}")
    vm02: dict[str, t.Any] = Field(serialization_alias=f"{vagrant_domains.vm02}")
    vm03: dict[str, t.Any] = Field(serialization_alias=f"{vagrant_domains.vm03}")
    vm04: dict[str, t.Any] = Field(serialization_alias=f"{vagrant_domains.vm04}")

    model_config = ConfigDict(frozen=True)

    def copy_and_add_values(self, other: "HostVars", *, allow_override: bool = True) -> "HostVars":
        """既存の値を維持しつつ、各VMに `other` の値を追加する

        Args:
            other (HostVars): _description_
            allow_override (bool, optional): _description_. Defaults to True.

        Raises:
            ValueError: _description_

        Returns:
            HostVars: _description_
        """
        self_vm_to_keys: dict[str, set[str]] = {vm_name: set(v.keys()) for vm_name, v in self}
        other_vm_to_keys: dict[str, set[str]] = {vm_name: set(v.keys()) for vm_name, v in other}

        if not allow_override:
            # 同じキーがあったらエラー
            for vm_name in self_vm_to_keys:
                if self_vm_to_keys[vm_name] & other_vm_to_keys[vm_name]:
                    raise ValueError(f"{vm_name=} has same keys")

        return HostVars(
            vm_dns={**self.vm_dns, **other.vm_dns},
            vm01={**self.vm01, **other.vm01},
            vm02={**self.vm02, **other.vm02},
            vm03={**self.vm03, **other.vm03},
            vm04={**self.vm04, **other.vm04},
        )


class Meta(BaseModel):
    # - VagrantHost members
    # - custom vars for each host
    hostvars: HostVars

    model_config = ConfigDict(frozen=True)


class InventoryOutputModel(BaseModel):
    dns_server: GroupModel
    k8s_cp_load_balancer: GroupModel
    k8s_cp_master: GroupModel
    k8s_other_nodes: GroupModel
    k8s_all: GroupModel
    all: GroupModel

    # Updated in VagrantInventory class
    vagrant_all: GroupModel

    # Updated in VagrantInventory class
    meta_info: Meta = Field(serialization_alias="_meta")

    model_config = ConfigDict(frozen=True)

    def list_all_hosts(self) -> set[str]:
        """`hosts`以下の名前一覧を返す

        例: 以下の場合は `{"vm01", "vm02", "vm03", "vm04"}` を返す

        ```yaml
        all:
          children:
            group1:
              hosts:
                - vm01
                - vm02
            group3:
              hosts:
                - vm03
                - vm04
        ```

        """
        out: set[str] = set()
        for _, v in self:
            match v:
                case GroupModel(hosts=hosts):
                    out |= set(hosts)
                    continue
                case _:
                    pass
        return out

    def copy_and_update_meta_info(self, meta_info: Meta, allow_override: bool = False) -> "InventoryOutputModel":
        c = self.model_copy(deep=True)
        return InventoryOutputModel(
            dns_server=c.dns_server,
            k8s_cp_load_balancer=c.k8s_cp_load_balancer,
            k8s_cp_master=c.k8s_cp_master,
            k8s_other_nodes=c.k8s_other_nodes,
            k8s_all=c.k8s_all,
            vagrant_all=c.vagrant_all,
            all=c.all,
            meta_info=Meta(
                hostvars=c.meta_info.hostvars.copy_and_add_values(
                    meta_info.hostvars,
                    allow_override=allow_override,
                )
            ),
        )


class ProvisioningVagrantInventory:
    def __init__(
        self,
        inventory_config: InventoryOutputModel,
    ) -> None:
        args = Args().parse_args()
        match (args.list, args.host):
            case (True, None):
                if not args.dry_run and inventory_config.list_all_hosts() != get_running_hosts():
                    raise ValueError("not all hosts are running")

                print(
                    insert_provisioning_all_vars(
                        inventory_config,
                        host=None,
                    ).model_dump_json(by_alias=True, exclude_none=True)
                )
            case (False, str(host)):
                if not args.dry_run and host not in get_running_hosts():
                    raise ValueError(f"{ host } is not running")
                json.dump(
                    inventory_config.meta_info.hostvars.model_dump().get(host, {}),
                    sys.stdout,
                )
            case _:
                raise ValueError("require either --list or --host <hostname>")


def get_running_hosts() -> set[Hostname]:
    """List running host names"""
    hosts: set[str] = set()

    cmd: str = "vagrant status --machine-readable"
    status: str = subprocess.check_output(cmd.split()).decode(sys.stdout.encoding)

    host: str
    key: str
    value: str
    for line in status.splitlines():
        (_, host, key, value) = line.split(",")[:4]
        match (key, value):
            case ("state", "running"):
                hosts.add(host)
            case _:
                pass
    return hosts


def insert_provisioning_all_vars(inventory_config: InventoryOutputModel, host: str | None) -> InventoryOutputModel:
    if host is None:
        # 全ての host の hostvars に追加
        vagrant_info = {h: get_host_vagrant_info(host=h) for h in get_running_hosts()}
    else:
        vagrant_info = {host: get_host_vagrant_info(host=host)}

    # `all.vars` 以下の `network_configs` を更新
    return InventoryOutputModel(
        dns_server=inventory_config.dns_server.model_copy(),
        k8s_cp_master=inventory_config.k8s_cp_master.model_copy(),
        k8s_cp_load_balancer=inventory_config.k8s_cp_load_balancer.model_copy(),
        k8s_other_nodes=inventory_config.k8s_other_nodes.model_copy(),
        k8s_all=inventory_config.k8s_all.model_copy(),
        vagrant_all=GroupModel(hosts=[h for h in get_running_hosts()]),
        all=inventory_config.all.model_copy(
            update=dict(
                vars={
                    "network_configs": create_network_configs_dns(vagrant_info),
                },
            )
        ),
        meta_info=inventory_config.meta_info.model_copy(),
    )


def get_host_vagrant_info(host: str) -> VagrantProvisioningInfo:
    """Get Provisioning Information from vagrant ssh-config"""
    cmd: str = f"vagrant ssh-config {host}"
    p = subprocess.Popen(cmd.split(), stdout=subprocess.PIPE)
    config: paramiko.SSHConfig = paramiko.SSHConfig()
    stdout: t.IO[bytes] | None = p.stdout
    if stdout is None:
        raise ValueError(f"{stdout=} is None")
    out: str = stdout.read().decode(sys.stdout.encoding)
    config.parse(io.StringIO("\n".join(out.splitlines()), newline="\n"))
    c: paramiko.SSHConfigDict = config.lookup(host)
    return VagrantProvisioningInfo(ipv4_address=c["hostname"])


def create_network_configs_dns(vagrant_info: dict[Hostname, VagrantProvisioningInfo]) -> dict[str, t.Any]:
    # last octet
    host_to_last_octet: dict[str, int] = {h: int(info.ipv4_address.split(".")[-1]) for h, info in vagrant_info.items()}
    network_component = "192.168.121"  # 192.168.121.0/24 : vagrant-libvirt default network
    # validation
    for host, addr in host_to_last_octet.items():
        assert 1 <= addr <= 254, f"{host=} {addr=} is not in range 1-254"
        assert (
            f"{network_component}.{addr}" == vagrant_info[host].ipv4_address
        ), f"{host=} {addr=} {vagrant_info[host].ipv4_address=}"

    # name server
    host_to_last_octet[vagrant_domains.ns1] = int(vagrant_info[f"{vagrant_domains.vm_dns}"].ipv4_address.split(".")[-1])
    return {
        "name_server": f"{network_component}.{host_to_last_octet[vagrant_domains.ns1]}",
        "dns": {
            "acl": {  # Access Control List
                "internal_network": [
                    "localhost",
                    f"{network_component}.0/24",
                ]
            },
            "domains": {
                f"{vagrant_domain}": {
                    "ipv4": [
                        {
                            "network_component": network_component,
                            "addresses": {
                                h.removesuffix(f".{vagrant_domain}"): last_octet
                                for h, last_octet in host_to_last_octet.items()
                            },
                        },
                    ],
                    "ipv6": [],
                    "cnames": {  # format is 'cname: actual'
                        "k8s-cp-endpoint": "vm-dns",  # kubernetes control plane endpoint (load_balancer)
                    },
                },
            },
        },
    }


def main() -> None:
    logging.basicConfig(
        format="[%(asctime)s][%(levelname)s][%(filename)s:%(lineno)d] - %(message)s",
        level=logging.WARNING,
    )
    ProvisioningVagrantInventory(
        inventory_config=InventoryOutputModel(
            dns_server=GroupModel(hosts=[f"{vagrant_domains.vm_dns}"]),
            k8s_cp_load_balancer=GroupModel(hosts=[f"{vagrant_domains.vm_dns}"]),
            k8s_cp_master=GroupModel(hosts=[f"{vagrant_domains.vm01}"]),
            k8s_other_nodes=GroupModel(
                hosts=[f"{vagrant_domains.vm02}", f"{vagrant_domains.vm03}", f"{vagrant_domains.vm04}"],
            ),
            k8s_all=GroupModel(
                children=[
                    "k8s_cp_load_balancer",
                    "k8s_cp_master",
                    "k8s_other_nodes",
                ],
                vars={
                    "k8s_cp_endpoint": f"{vagrant_domains.k8s_cp_endpoint}",
                },
            ),
            vagrant_all=GroupModel(),
            all=GroupModel(),
            meta_info=Meta(
                hostvars=HostVars(
                    vm_dns={},
                    vm01={},
                    vm02={
                        "k8s_is_control_plane": True,
                    },
                    vm03={
                        "k8s_is_control_plane": False,
                    },
                    vm04={
                        "k8s_is_control_plane": False,
                    },
                ),
            ),
        ),
    )


if __name__ == "__main__":
    main()
