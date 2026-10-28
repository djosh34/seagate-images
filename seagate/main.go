// Command seagate runs the steps of the Seagate pods: connect, passwd and backup.
package main

import (
	"fmt"
	"log"
	"os"
)

const usage = `usage:
  seagate connect <option>...  write the options as one connect line to /dev/nvme-fabrics, then wait
  seagate passwd               copy /secret/smbpasswd to /state/smbpasswd
  seagate backup               snapshot the disk at /disk, back it up with restic, report to Home Assistant`

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "connect":
		err = runConnect(os.Args[2:])
	case "passwd":
		err = runPasswd(os.Args[2:])
	case "backup":
		err = runBackup(os.Args[2:])
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		log.Printf("seagate %s: %v", os.Args[1], err)
		os.Exit(1)
	}
}
