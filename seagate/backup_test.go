package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// fake records every action of a run and fails the one named in fail.
type fake struct {
	calls []string
	fail  string
}

func (f *fake) nightly() nightly {
	do := func(call string) error {
		f.calls = append(f.calls, call)
		if call == f.fail {
			return errors.New("boom")
		}
		return nil
	}
	return nightly{
		repository: "rclone:box:",
		restic:     func(args ...string) error { return do("restic " + strings.Join(args, " ")) },
		snapshot:   func(src, dst string) error { return do("snapshot " + src + " " + dst) },
		remove:     func(path string) error { return do("remove " + path) },
		post:       func(body map[string]string) error { return do(fmt.Sprint("post ", body)) },
	}
}

// repoCalls are the restic calls for one repository, in order.
func repoCalls(name, keep string) []string {
	flags := " --repo rclone:box:" + name + " --cache-dir /disk/restic-cache -o rclone.program=/rclone"
	return []string{
		"restic unlock" + flags,
		"restic backup /disk/snapshots/" + name + " --host seagate" + flags,
		"restic forget --host seagate --prune " + keep + flags,
		"restic check" + flags,
	}
}

func TestBackupStepOrder(t *testing.T) {
	f := &fake{}
	if err := f.nightly().run(); err != nil {
		t.Fatal(err)
	}
	want := slices.Concat(
		[]string{
			"remove /disk/snapshots/timemachine",
			"remove /disk/snapshots/data",
			"snapshot /disk/timemachine /disk/snapshots/timemachine",
			"snapshot /disk/data /disk/snapshots/data",
		},
		repoCalls("timemachine", "--keep-daily 7 --keep-weekly 4"),
		repoCalls("data", "--keep-daily 7 --keep-weekly 4 --keep-monthly 12"),
		[]string{
			"remove /disk/snapshots/timemachine",
			"remove /disk/snapshots/data",
			"post map[status:ok]",
		},
	)
	if !slices.Equal(f.calls, want) {
		t.Errorf("calls:\n%s\nwant:\n%s", strings.Join(f.calls, "\n"), strings.Join(want, "\n"))
	}
}

func TestBackupFailedStep(t *testing.T) {
	failing := repoCalls("timemachine", "--keep-daily 7 --keep-weekly 4")[2]
	f := &fake{fail: failing}
	err := f.nightly().run()
	if err == nil || !strings.HasPrefix(err.Error(), "forget timemachine: ") {
		t.Fatalf("err = %v, want the failed step forget timemachine", err)
	}
	i := slices.Index(f.calls, failing)
	want := []string{
		failing,
		"remove /disk/snapshots/timemachine",
		"remove /disk/snapshots/data",
		"post map[status:failed step:forget timemachine]",
	}
	if i < 0 || !slices.Equal(f.calls[i:], want) {
		t.Errorf("calls:\n%s\nwant to end with:\n%s", strings.Join(f.calls, "\n"), strings.Join(want, "\n"))
	}
}

func TestBackupFailedSnapshot(t *testing.T) {
	f := &fake{fail: "snapshot /disk/data /disk/snapshots/data"}
	if err := f.nightly().run(); err == nil {
		t.Fatal("run succeeded, want an error")
	}
	want := []string{
		"remove /disk/snapshots/timemachine",
		"remove /disk/snapshots/data",
		"snapshot /disk/timemachine /disk/snapshots/timemachine",
		"snapshot /disk/data /disk/snapshots/data",
		"remove /disk/snapshots/timemachine",
		"remove /disk/snapshots/data",
		"post map[status:failed step:snapshot data]",
	}
	if !slices.Equal(f.calls, want) {
		t.Errorf("calls:\n%s\nwant:\n%s", strings.Join(f.calls, "\n"), strings.Join(want, "\n"))
	}
}
