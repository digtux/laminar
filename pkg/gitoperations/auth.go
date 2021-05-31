package gitoperations

import (
	"go.uber.org/zap"
	"io/ioutil"

	"github.com/digtux/laminar/pkg/common"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	crypto_ssh "golang.org/x/crypto/ssh"
)

func getSshKeySigner(fileName string, log *zap.SugaredLogger) crypto_ssh.Signer {

	fullPath := common.GetFileAbsPath(fileName, log)
	sshKey, err := ioutil.ReadFile(fullPath)
	if err != nil {
		log.Fatalw("unable to read private ssh key",
			"file", fileName,
			"error", err,
		)
	}

	signer, err := crypto_ssh.ParsePrivateKey([]byte(sshKey))
	if err != nil {
		log.Fatalw("Failed to parse ssh key",
			"action", "sshKeyParse",
			"sshKey", fileName,
			"error", err.Error(),
		)
	}
	return signer
}

func getAuth(key string, log *zap.SugaredLogger) *ssh.PublicKeys {
	signer := getSshKeySigner(key, log)
	auth := &ssh.PublicKeys{
		User:   "git",
		Signer: signer,
	}
	auth.HostKeyCallback = crypto_ssh.InsecureIgnoreHostKey()
	return auth
}
