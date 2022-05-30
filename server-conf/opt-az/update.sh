#!/usr/bin/env bash
# cd /opt/az/antizapret-pac-generator-light

timeout 30 git pull --ff-only

set -euxo pipefail

./update.sh
./parse.sh

if ! diff -q result/knot-aliases-alt.conf /etc/knot-resolver/az-aliases.lua; then
    cp result/knot-aliases-alt.conf /etc/knot-resolver/az-aliases.lua
    echo 'cache.clear()' | socat - UNIX-CONNECT:/run/knot-resolver/control/1 || :
    systemctl restart kresd@1.service
fi

birdconvert() {
    sort -V "$1" | sed -e 's/^/route /; s/$/ recursive 0.0.0.0;/;'
}

if ! diff <(birdconvert result/blocked-ranges.txt) /etc/bird/az_routes.conf; then
    birdconvert result/blocked-ranges.txt >/etc/bird/az_routes.conf
    echo configure | birdc
fi
