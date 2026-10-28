package main

import (
	"errors"
	"os"
)

// runPasswd copies the smbpasswd file from the Secret into Samba's state
// folder. Samba wants to own the file at mode 0600, which a Secret mount
// cannot give.
func runPasswd(args []string) error {
	if len(args) != 0 {
		return errors.New("passwd takes no arguments")
	}
	data, err := os.ReadFile("/secret/smbpasswd")
	if err != nil {
		return err
	}
	return os.WriteFile("/state/smbpasswd", data, 0o600)
}
