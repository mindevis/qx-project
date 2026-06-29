package deploy

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

func parseSSHSigner(pem []byte, passphrase string) (ssh.Signer, error) {
	if len(passphrase) > 0 {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(pem, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidSSHKey, err)
		}
		return signer, nil
	}
	signer, err := ssh.ParsePrivateKey(pem)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSSHKey, err)
	}
	return signer, nil
}
