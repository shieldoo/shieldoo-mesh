#!/bin/sh

# catch terminate signals
#trap 'exit 0' SIGTERM

echo '
       .__    .__       .__       .___             
  _____|  |__ |__| ____ |  |    __| _/____   ____  
 /  ___/  |  \|  |/ __ \|  |   / __ |/  _ \ /  _ \ 
 \___ \|   Y  \  \  ___/|  |__/ /_/ (  <_> |  <_> )
/____  >___|  /__|\___  >____/\____ |\____/ \____/ 
     \/     \/        \/           \/             '
echo ''

mkdir -p /dev/net
if [ ! -c /dev/net/tun ]; then
    mknod /dev/net/tun c 10 200
fi

echo ''
echo 'starting shieldoo ..'

exec /app/shieldoo-mesh-srv -run