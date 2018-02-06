# -*- mode: ruby -*-
# vi: set ft=ruby :

raise 'GOPATH must be defined' unless ENV.key? 'GOPATH'

$script = <<SCRIPT
apt-get update
apt-get -y install awscli make gcc
mkdir -p /home/ubuntu/bin
curl -sL -o /home/ubuntu/bin/gimme https://raw.githubusercontent.com/travis-ci/gimme/master/gimme
chmod +x /home/ubuntu/bin/gimme
chown -R ubuntu:ubuntu /home/ubuntu/bin
SCRIPT

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/xenial64"
  config.vm.synced_folder ENV['GOPATH'], "/home/ubuntu/go"
  config.vm.synced_folder "./example", "/home/ubuntu/awsbeats-example"
  config.vm.provider "virtualbox" do |v|
    v.memory = 1024
    v.cpus = 1
  end
  config.vm.provision "shell", inline: $script
end
