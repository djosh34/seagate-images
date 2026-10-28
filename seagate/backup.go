package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"
)

const (
	disk      = "/disk"
	snapshots = disk + "/snapshots"
)

// repositories are backed up in this order. Each one backs up the snapshot of
// the subvolume with the same name.
var repositories = []struct {
	name string
	keep []string
}{
	{"timemachine", []string{"--keep-daily", "7", "--keep-weekly", "4"}},
	{"data", []string{"--keep-daily", "7", "--keep-weekly", "4", "--keep-monthly", "12"}},
}

// nightly holds the actions of one backup run, so tests can replace them.
type nightly struct {
	repository string // restic repository prefix, the repository name is appended
	restic     func(args ...string) error
	snapshot   func(src, dst string) error
	remove     func(path string) error
	post       func(body map[string]string) error
}

// runBackup runs the nightly backup. It reads the restic repository prefix
// and the Home Assistant webhook URL from its environment. restic reads its
// password and the rclone settings from the environment too.
func runBackup(args []string) error {
	if len(args) != 0 {
		return errors.New("backup takes no arguments")
	}
	repository := os.Getenv("SEAGATE_REPOSITORY")
	webhook := os.Getenv("SEAGATE_WEBHOOK_URL")
	if repository == "" || webhook == "" {
		return errors.New("SEAGATE_REPOSITORY and SEAGATE_WEBHOOK_URL must be set")
	}
	return nightly{
		repository: repository,
		restic:     runRestic,
		snapshot:   btrfsSnapshot,
		remove:     os.RemoveAll,
		post:       func(body map[string]string) error { return postJSON(webhook, body) },
	}.run()
}

// run takes the snapshots, backs them up, deletes them and reports the
// result. After a failed step it runs no further restic step, but still
// deletes the snapshots before it reports the failure.
func (n nightly) run() error {
	step, err := n.steps()
	if removeErr := n.removeSnapshots(); removeErr != nil {
		if err == nil {
			step, err = "delete snapshots", removeErr
		} else {
			log.Printf("delete snapshots: %v", removeErr)
		}
	}
	if err != nil {
		err = fmt.Errorf("%s: %w", step, err)
		if postErr := n.post(map[string]string{"status": "failed", "step": step}); postErr != nil {
			return errors.Join(err, fmt.Errorf("report: %w", postErr))
		}
		return err
	}
	if err := n.post(map[string]string{"status": "ok"}); err != nil {
		return fmt.Errorf("report: %w", err)
	}
	return nil
}

// steps runs every step up to the first failure and returns that step's name.
func (n nightly) steps() (string, error) {
	log.Print("delete old snapshots")
	if err := n.removeSnapshots(); err != nil {
		return "delete old snapshots", err
	}
	for _, r := range repositories {
		step := "snapshot " + r.name
		log.Print(step)
		if err := n.snapshot(disk+"/"+r.name, snapshots+"/"+r.name); err != nil {
			return step, err
		}
	}
	for _, r := range repositories {
		forget := append([]string{"forget", "--host", "seagate", "--prune"}, r.keep...)
		for _, args := range [][]string{
			{"unlock"},
			{"backup", snapshots + "/" + r.name, "--host", "seagate"},
			forget,
			{"check"},
		} {
			step := args[0] + " " + r.name
			log.Print(step)
			args = append(args, "--repo", n.repository+r.name, "--cache-dir", disk+"/restic-cache", "-o", "rclone.program=/rclone")
			if err := n.restic(args...); err != nil {
				return step, err
			}
		}
	}
	return "", nil
}

// removeSnapshots deletes both snapshots. A missing snapshot is not an error.
// The snapshots are writable, so a plain recursive delete removes them.
func (n nightly) removeSnapshots() error {
	var errs []error
	for _, r := range repositories {
		errs = append(errs, n.remove(snapshots+"/"+r.name))
	}
	return errors.Join(errs...)
}

func runRestic(args ...string) error {
	cmd := exec.Command("/restic", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func postJSON(webhook string, body map[string]string) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	client := http.Client{Timeout: time.Minute}
	resp, err := client.Post(webhook, "application/json", bytes.NewReader(data))
	// The URL holds the webhook's secret ID, so it stays out of the log.
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("post to the webhook: %w", urlErr.Err)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("webhook answered %s", resp.Status)
	}
	return nil
}
