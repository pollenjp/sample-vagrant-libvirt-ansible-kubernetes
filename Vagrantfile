# frozen_string_literal: true

# VM Spec Data
class VmSpecData
  attr_reader :name, :cpu, :memory, :box, :comment, :provider

  # rubocop:disable Metrics/ParameterLists
  def initialize(name, cpu, memory, box, provider, comment)
    # rubocop:enable Metrics/ParameterLists
    @name = name
    @cpu = cpu
    @memory = memory
    @box = box
    @provider = provider
    @comment = comment
  end
end

VAGRANT_BOX = 'generic/ubuntu2204'
VM_SPEC_ARR = [
  VmSpecData.new('vm-dns.vagrant.home', 2, 2048, VAGRANT_BOX, 'libvirt', 'dns node'),
  VmSpecData.new('vm01.vagrant.home', 4, 4096, VAGRANT_BOX, 'libvirt', 'cp01 node'),
  VmSpecData.new('vm02.vagrant.home', 2, 4096, VAGRANT_BOX, 'libvirt', 'cp02 node'),
  VmSpecData.new('vm03.vagrant.home', 2, 2048, VAGRANT_BOX, 'libvirt', 'worker01 node'),
  VmSpecData.new('vm04.vagrant.home', 2, 2048, VAGRANT_BOX, 'libvirt', 'worker02 node')
].freeze

Vagrant.configure('2') do |config|
  VM_SPEC_ARR.each do |spec|
    config.vm.define spec.name do |v|
      v.vm.box = spec.box
      v.vm.hostname = spec.name
      # v.vm.network 'private_network', type: 'dhcp'
      v.vm.provider spec.provider do |prov|
        # https://vagrant-libvirt.github.io/vagrant-libvirt/configuration.html
        prov.cpus = spec.cpu
        prov.memory = spec.memory

        # https://vagrant-libvirt.github.io/vagrant-libvirt/configuration.html#management-network
        # libvirt.management_network_name = 'my_network'
        # libvirt.management_network_address = '192.168.121.0/24'
      end
    end
  end
end
