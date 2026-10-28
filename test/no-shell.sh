#!/bin/bash
# Fails if the image holds a shell, a package manager, an interpreter or cp.
set -euo pipefail
image=$1
id=$(docker create "$image")
files=$(docker export "$id" | tar -t)
docker rm "$id" >/dev/null
if grep -E '(^|/)(sh|bash|ash|dash|busybox|python[0-9.]*|perl[0-9.]*|dnf[0-9]*|rpm|apk|cp)$' <<<"$files"; then
  echo "$image holds the programs above" >&2
  exit 1
fi
echo "$image: $(wc -l <<<"$files") files, no shell"
