#!/bin/bash

echo "Installing shieldoo-mesh-svc"

# create directory
mkdir -p /opt/shieldoo-mesh

# uninstall if already installed
if [ -f /etc/init.d/shieldoo-mesh ] || [ -f /etc/systemd/system/shieldoo-mesh.service ]; then
        /opt/shieldoo-mesh/shieldoo-mesh-srv -service stop
        /opt/shieldoo-mesh/shieldoo-mesh-srv -service uninstall
fi

# copy new files
wget -qO- "https://download.shieldoo.io/latest/linux-amd64-shieldoo-mesh-svc-setup.tar.gz" | tar -xvz -C /opt/shieldoo-mesh

chmod 755 /opt/shieldoo-mesh/shieldoo-mesh-srv

# install service and configuroation data
/opt/shieldoo-mesh/shieldoo-mesh-srv -createconfig "$1"
/opt/shieldoo-mesh/shieldoo-mesh-srv -service install
/opt/shieldoo-mesh/shieldoo-mesh-srv -service start
