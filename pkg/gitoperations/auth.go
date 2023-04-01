package gitoperations

import (
	"os"

	"github.com/digtux/laminar/pkg/common"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	cryptossh "golang.org/x/crypto/ssh"
)

func (c *Client) getSSHKeySigner(fileName string) cryptossh.Signer {
	fullPath := common.GetFileAbsPath(fileName, c.logger)
	sshKey, err := os.ReadFile(fullPath)
	if err != nil {
		c.logger.Fatalw("unable to read private ssh key",
			"file", fileName,
			"error", err,
		)
	}

	signer, err := cryptossh.ParsePrivateKey(sshKey)
	if err != nil {
		c.logger.Fatalw("Failed to parse ssh key",
			"action", "sshKeyParse",
			"sshKey", fileName,
			"error", err.Error(),
		)
	}
	return signer
}

func (c *Client) getAuth(key string) *ssh.PublicKeys {
	signer := c.getSSHKeySigner(key)
	auth := &ssh.PublicKeys{
		User:   "git",
		Signer: signer,
	}
	// auth.HostKeyCallback = cryptossh.InsecureIgnoreHostKey()
	return auth
}
