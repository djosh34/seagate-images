#!/bin/bash
# Copies smbd, samba-dcerpcd with its helpers, the modules we load, every
# shared library they need, /etc/passwd and /etc/group into $1. /tmp stays
# empty: smbd only needs it to exist, as the path of the IPC$ share.
set -euo pipefail
root=$1
files=(
  /usr/sbin/smbd
  /usr/libexec/samba/samba-dcerpcd
  /usr/libexec/samba/rpcd_*
  # Loaded with dlopen, so ldd of smbd does not show them.
  /usr/lib64/samba/vfs/catia.so
  /usr/lib64/samba/vfs/fruit.so
  /usr/lib64/samba/vfs/streams_xattr.so
  /usr/lib64/samba/pdb/smbpasswd.so
)
mkdir -p "$root"/etc "$root"/tmp
for f in "${files[@]}"; do
  cp -a --parents "$f" "$root"
  ldd "$f" | awk '/=> \//{print $3} /^\t\/lib/{print $1}'
done | sort -u | while read -r lib; do
  cp -L --parents "$lib" "$root"
done
cp /passwd /group "$root"/etc/
