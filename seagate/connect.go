package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const fabricsDevice = "/dev/nvme-fabrics"

// runConnect connects the NVMe/TCP controller once and then waits until it is
// stopped. The kernel keeps the controller and reconnects it by itself.
func runConnect(args []string) error {
	if len(args) == 0 {
		return errors.New("no connect options given")
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	device, err := os.OpenFile(fabricsDevice, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	err = connect(device, strings.Join(args, ","))
	if closeErr := device.Close(); err == nil && closeErr != nil {
		err = closeErr
	}
	if err != nil {
		return err
	}

	log.Print("connected, waiting until stopped")
	<-stop
	return nil
}

// connect writes one connect line. The kernel answers EALREADY when the
// controller still exists, for example after a pod restart. That counts as
// connected.
func connect(device io.Writer, line string) error {
	_, err := io.WriteString(device, line)
	if errors.Is(err, syscall.EALREADY) {
		log.Print("the controller already exists")
		return nil
	}
	if err != nil {
		return fmt.Errorf("connect %s: %w", line, err)
	}
	return nil
}
