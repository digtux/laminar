package cfg

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func debugLogger() (log *zap.SugaredLogger) {
	zapLogger, _ := zap.NewDevelopment()
	sugar := zapLogger.Sugar()
	return sugar
}

func TestParseConfig(t *testing.T) {

	log := debugLogger()
	testData := []byte(`---
global:
  gitUser: Laminar
  gitEmail: laminar@example.org
  gitMessage: automated promotion

dockerRegistries:
- reg: gcr.io/orgname
  name: gcr
- reg: 112233445566.dkr.ecr.eu-west-2.amazonaws.com/orgname
  name: ecr

git:
- name: monorepo1
  url: git@github.com:digtux/ray-example-kapitan.git
  branch: feature/foo
  key: ~/example_ssh_id_rsa
  updates:

  - pattern: "glob:master-*"
    files:
    - path: inventory/classes/images-staging.yml

- name: monorepo2
  url: git@github.com:digtux/ray-example-kapitan.git
  branch: master
  key: ~/example_ssh_id_rsa
  pollFreq: 30
  remoteConfig: true
  updates:

  - pattern: "glob:develop-*"
    files:
    - path: inventory/classes/images-dev.yml

  - pattern: "glob:master-*"
    files:
    - path: inventory/classes/images-staging.yml

  - pattern: "glob:release-*"
    files:
    - path: inventory/classes/images-release.yml
`)
	result, err := parseConfig(testData, log)
	if err != nil {
		fmt.Println(err)
	}

	if len(result.DockerRegistries) < 0 {
		t.Error("unable to see any configured registries")
	}

	assert.Equal(t, ":8080", result.Global.Listener)
	assert.Equal(t, 60, result.GitRepos[0].PollFreq)
	assert.Equal(t, 30, result.GitRepos[1].PollFreq)
	assert.Equal(t, "feature/foo", result.GitRepos[0].Branch)
	assert.Equal(t, "master", result.GitRepos[1].Branch)
	assert.Equal(t, 2, len(result.GitRepos))
	assert.Equal(t, 1, len(result.GitRepos[0].Updates))
	assert.Equal(t, 3, len(result.GitRepos[1].Updates))
	assert.Equal(t, false, result.GitRepos[0].RemoteConfig)
	assert.Equal(t, true, result.GitRepos[1].RemoteConfig)

}
