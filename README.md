# seagate-images

Two shell-free images for the Seagate disk on the Talos node of [djosh34/talos_home_assistent](https://github.com/djosh34/talos_home_assistent). Both are `FROM scratch`, linux/amd64 only, and public:

- `ghcr.io/djosh34/seagate-samba`: smbd, samba-dcerpcd with its rpcd helpers, the vfs modules catia, fruit and streams_xattr, the smbpasswd backend and their libraries, copied from a Fedora 44 stage. Users smb (2000), tm (2001), rw (2002) and nobody. It runs as 2000. smb.conf comes from the pod.
- `ghcr.io/djosh34/seagate-tools`: restic at `/restic`, rclone at `/rclone` (static release binaries, checksums checked) and the Go program `/seagate`.

Neither image holds a shell, a package manager, an interpreter or cp. CI checks that.

## seagate

- `seagate connect <option>...` joins the options with commas and writes them as one connect line to `/dev/nvme-fabrics`, then waits until it is stopped. If the controller still exists, for example after a pod restart, the kernel answers EALREADY, which counts as connected.
- `seagate passwd` copies `/secret/smbpasswd` to `/state/smbpasswd`. Samba must own that file at mode 0600, which a Secret mount cannot give.
- `seagate backup` runs the nightly backup of the disk mounted at `/disk`:
  1. Delete `/disk/snapshots/timemachine` and `/disk/snapshots/data` if they exist.
  2. Snapshot the subvolumes `/disk/timemachine` and `/disk/data` into `/disk/snapshots`. The snapshots are writable.
  3. For the repositories timemachine and then data: `restic unlock`, `backup --host seagate`, `forget --host seagate --prune` (timemachine keeps 7 daily and 4 weekly, data also 12 monthly), `check`.
  4. Delete both snapshots.
  5. POST `{"status": "ok"}` to the Home Assistant webhook.

  If a step fails, it runs no further restic step, deletes both snapshots, POSTs `{"status": "failed", "step": "<step>"}` and exits non-zero.

  Environment: `SEAGATE_REPOSITORY` is the restic repository prefix, for example `rclone:storagebox:`, and the repository name is appended. `SEAGATE_WEBHOOK_URL` is the webhook. restic reads its password (`RESTIC_PASSWORD_FILE`) and rclone its remote (`RCLONE_CONFIG_*`) from the environment too. Every restic call uses `--cache-dir /disk/restic-cache` and `-o rclone.program=/rclone`. restic writes temporary packs to `/tmp`, so the pod mounts an emptyDir there.

## CI

`.github/workflows/images.yml` runs on every pull request and on main:

- `go vet` and the unit tests, then the snapshot test on a loop-mounted btrfs.
- Builds both images and fails if one holds a shell (`test/no-shell.sh`).
- Starts smbd as uid 2000 with no capabilities and a read-only root, and logs in with smbclient. A wrong password must fail (`test/samba.sh`).
- On main only, pushes both images as `latest` and as the commit SHA. Pin them by digest.

Locally: `go test ./...`.
