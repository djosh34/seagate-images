package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// TestBtrfsSnapshot needs a btrfs directory that holds a subvolume named
// subvolume and a folder named snapshots, both owned by the test's user.
// CI sets it up. Without SEAGATE_BTRFS_DIR the test is skipped.
func TestBtrfsSnapshot(t *testing.T) {
	dir := os.Getenv("SEAGATE_BTRFS_DIR")
	if dir == "" {
		t.Skip("SEAGATE_BTRFS_DIR is not set")
	}
	src := filepath.Join(dir, "subvolume")
	dst := filepath.Join(dir, "snapshots", "subvolume")
	if err := os.WriteFile(filepath.Join(src, "file"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := btrfsSnapshot(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "file"))
	if err != nil || string(data) != "data" {
		t.Fatalf("file in the snapshot: %q, %v", data, err)
	}
	// The root of every btrfs subvolume has inode 256.
	var st syscall.Stat_t
	if err := syscall.Stat(dst, &st); err != nil || st.Ino != 256 {
		t.Fatalf("snapshot inode %d, %v, want a subvolume with inode 256", st.Ino, err)
	}
	if err := os.WriteFile(filepath.Join(dst, "file"), []byte("changed"), 0o600); err != nil {
		t.Fatalf("snapshot is not writable: %v", err)
	}
	if err := os.RemoveAll(dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("snapshot still there after the delete: %v", err)
	}
}
