/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package ssh

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// GetHostKey checks the local user's known_hosts file
// If the file doesn't exist, an error is returned
// If the desired entry does not exist, an error including remediation steps is returned
func GetHostKey(host string) (ssh.PublicKey, error) {
	// ~/.ssh/known_hosts
	file, err := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		return nil, errors.New("file does not exist")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var hostKey ssh.PublicKey
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) != 3 {
			continue
		}
		if strings.Contains(fields[0], host) {
			var err error

			hostKey, _, _, _, err = ssh.ParseAuthorizedKey(scanner.Bytes())
			if err != nil {
				return nil, fmt.Errorf("error parsing %q: %v", fields[2], err)
			}
			break
		}
	}
	if hostKey == nil {
		return nil, fmt.Errorf("no hostkey found for %s - please run `ssh-keyscan -H %s >> ~/.ssh/known_hosts` to remedy", host, host)
	}

	return hostKey, nil
}
