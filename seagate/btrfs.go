package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// BTRFS_IOC_SNAP_CREATE from linux/btrfs.h: _IOW(0x94, 1, struct btrfs_ioctl_vol_args).
const btrfsIocSnapCreate = 0x50009401

// btrfsVolArgs is struct btrfs_ioctl_vol_args from linux/btrfs.h.
type btrfsVolArgs struct {
	fd   int64
	name [4088]byte
}

// btrfsSnapshot makes a writable snapshot of the subvolume src at dst. The
// owner of src needs no capability for this.
func btrfsSnapshot(src, dst string) error {
	var args btrfsVolArgs
	name := filepath.Base(dst)
	if len(name) >= len(args.name) {
		return fmt.Errorf("snapshot name %s is too long", name)
	}
	copy(args.name[:], name)

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	parent, err := os.Open(filepath.Dir(dst))
	if err != nil {
		return err
	}
	defer parent.Close()

	args.fd = int64(source.Fd())
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, parent.Fd(), btrfsIocSnapCreate, uintptr(unsafe.Pointer(&args)))
	if errno != 0 {
		return fmt.Errorf("snapshot %s to %s: %w", src, dst, errno)
	}
	return nil
}
