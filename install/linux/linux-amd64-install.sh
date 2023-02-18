#!/bin/bash

echo "Starting the installation of Shieldoo Secure Network."

# create directory
mkdir -p /opt/shieldoo-mesh

# uninstall if already installed
if [ -f /etc/init.d/shieldoo-mesh ] || [ -f /etc/systemd/system/shieldoo-mesh.service ]; then
	echo "Uninstalling the previous version."
        /opt/shieldoo-mesh/shieldoo-mesh-srv -desktop -service stop
        /opt/shieldoo-mesh/shieldoo-mesh-srv -desktop -service uninstall
fi

# copy new files
echo "Getting the latest version."
wget -qO- "https://download.shieldoo.io/latest/linux-amd64-shieldoo-mesh-setup.tar.gz" | tar -xz -C /opt/shieldoo-mesh

chmod 755 /opt/shieldoo-mesh/shieldoo-mesh-srv
chmod 755 /opt/shieldoo-mesh/shieldoo-mesh-app

# install service and configuration data
echo "Installing."
/opt/shieldoo-mesh/shieldoo-mesh-srv -desktop -service install
/opt/shieldoo-mesh/shieldoo-mesh-srv -desktop -service start

# get owner
OWNER="$1"

# prepare config directory
su - "$OWNER" -c 'mkdir -p ~/.shieldoo'
MYCFG="uri: $2"
CFGEXISTS=$(su - $OWNER -c "[[ -f ~/.shieldoo/shieldoo-mesh.yaml ]] && echo 1 || echo 0")
if [ "$CFGEXISTS" == "1" ]; then
    su - "$OWNER" -c "cat ~/.shieldoo/shieldoo-mesh.yaml | grep -v 'uri: http' > ~/.shieldoo/shieldoo-mesh.yaml.tmp"
    su - "$OWNER" -c "cat ~/.shieldoo/shieldoo-mesh.yaml.tmp > ~/.shieldoo/shieldoo-mesh.yaml"
    su - "$OWNER" -c "echo \"${MYCFG}\" >> ~/.shieldoo/shieldoo-mesh.yaml"
else
    su - "$OWNER" -c "echo \"${MYCFG}\" > ~/.shieldoo/shieldoo-mesh.yaml"
fi


# create desktop icon
echo "[Desktop Entry]
Version=1.0
Type=Application
Terminal=false
Exec=/opt/shieldoo-mesh/shieldoo-mesh-app
Name=Shieldoo Secure Network
Comment=Shieldoo Secure Network Desktop Client
Icon=/opt/shieldoo-mesh/logo.png
" > /usr/share/applications/ShieldooMesh.desktop

echo "The Shieldoo Secure Network installation has been successfully completed."
