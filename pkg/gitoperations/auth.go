package gitoperations

import (
	"io/ioutil"

	"github.com/digtux/laminar/pkg/common"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	crypto_ssh "golang.org/x/crypto/ssh"
)

func (c *Client) getSshKeySigner(fileName string) crypto_ssh.Signer {

	fullPath := common.GetFileAbsPath(fileName, c.logger)
	sshKey, err := ioutil.ReadFile(fullPath)
	if err != nil {
		c.logger.Fatalw("unable to read private ssh key",
			"file", fileName,
			"error", err,
		)
	}

	signer, err := crypto_ssh.ParsePrivateKey([]byte(sshKey))
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
	signer := c.getSshKeySigner(key)
	auth := &ssh.PublicKeys{
		User:   "git",
		Signer: signer,
	}
	auth.HostKeyCallback = crypto_ssh.InsecureIgnoreHostKey()
	return auth
}
