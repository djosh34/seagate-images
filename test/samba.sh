#!/bin/bash
# Starts smbd from the samba image as uid 2000, with no capabilities and a
# read-only root, and logs in with the host's smbclient. `seagate passwd` from
# the tools image puts the password file in place first, as the init
# container will. Needs sudo and smbclient on the host.
set -euo pipefail
samba=$1
tools=$2
test=$(cd "$(dirname "$0")" && pwd)
dir=$(mktemp -d)
mkdir "$dir"/state "$dir"/timemachine "$dir"/data
sudo chown 2000:2000 "$dir"/state "$dir"/timemachine "$dir"/data
locked=(--user 2000:2000 --cap-drop ALL --security-opt no-new-privileges --read-only)

docker run --rm "${locked[@]}" \
  -v "$test"/smbpasswd:/secret/smbpasswd:ro -v "$dir"/state:/state \
  "$tools" passwd

docker run -d --name smbd "${locked[@]}" \
  --sysctl net.ipv4.ip_unprivileged_port_start=445 -p 127.0.0.1:445:445 \
  -v "$test"/smb.conf:/etc/samba/smb.conf:ro -v "$dir"/state:/state \
  -v "$dir"/timemachine:/timemachine -v "$dir"/data:/data \
  "$samba"
trap 'docker logs smbd; docker rm -f smbd >/dev/null' EXIT

echo hello > "$dir"/hello.txt
for i in $(seq 30); do
  smbclient -L //127.0.0.1 -U tm%seagate-test && break
  [ "$i" = 30 ] && exit 1
  sleep 1
done
smbclient //127.0.0.1/timemachine -U tm%seagate-test \
  -c "put $dir/hello.txt hello.txt; put $dir/hello.txt hello.txt:stream; allinfo hello.txt" |
  tee /dev/stderr | grep -F '[:stream:'
smbclient //127.0.0.1/data -U rw%seagate-test -c "put $dir/hello.txt hello.txt; ls"
sudo cmp "$dir"/hello.txt "$dir"/timemachine/hello.txt
sudo cmp "$dir"/hello.txt "$dir"/data/hello.txt

if out=$(smbclient //127.0.0.1/timemachine -U tm%wrong-password -c ls 2>&1); then
  echo "a wrong password logged in" >&2
  exit 1
fi
grep NT_STATUS_LOGON_FAILURE <<<"$out"
echo "samba: logins work, a wrong password fails"
