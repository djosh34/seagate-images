package main

import (
	"os"
	"syscall"
	"testing"
)

// fabrics fails every write with err, as /dev/nvme-fabrics does.
type fabrics struct{ err error }

func (f fabrics) Write(p []byte) (int, error) {
	return 0, &os.PathError{Op: "write", Path: fabricsDevice, Err: f.err}
}

func TestConnectAlreadyConnected(t *testing.T) {
	if err := connect(fabrics{syscall.EALREADY}, "transport=tcp"); err != nil {
		t.Errorf("EALREADY: %v, want connected", err)
	}
}

func TestConnectFails(t *testing.T) {
	if err := connect(fabrics{syscall.EINVAL}, "transport=tcp"); err == nil {
		t.Error("EINVAL: connected, want an error")
	}
}
