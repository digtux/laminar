package cfg

import (
	"fmt"
	"go.uber.org/zap"
	"reflect"
	"strings"
	"testing"
)

func debugLogger() (log *zap.SugaredLogger) {
	zapLogger, _ := zap.NewDevelopment()
	sugar := zapLogger.Sugar()
	return sugar
}

func TestParseConfig(t *testing.T) {

	testData := []byte(`---
global:
  gitUser: Laminar
  gitEmail: laminar@example.org
  gitMessage: automated promotion
  gitHubToken: somethingrandom

dockerRegistries:
- reg: gcr.io/orgname
  name: gcr
- reg: 112233445566.dkr.ecr.eu-west-2.amazonaws.com/orgname
  name: ecr

git:
- name: monorepo1
  url: git@github.com:digtux/ray-example-kapitan.git
  branch: master
  key: ~/example_ssh_id_rsa
  pollFreq: 60
  remoteConfig: false
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
	result, err := ParseConfig(testData)
	if err != nil {
		fmt.Println(err)
	}

	if result.Global.GitHubToken != "somethingrandom" {
		t.Error("wasn't able to read the global.gitHubToken")
	}

	if len(result.DockerRegistries) < 0 {
		t.Error("unable to see any configured registries")
	}

	if result.GitRepos[0].Branch != "master" {
		t.Errorf("unable to read branch from config, got: %s, expected: %s", result.GitRepos[0].Branch, "master")
	}

}
func TestParseConfigFailure(t *testing.T) {

	testData := []byte(`...garbage...`)
	empty := Config{}
	result, err := ParseConfig(testData)

	if !reflect.DeepEqual(result, empty) {
		t.Error("expected to load an empty struct")
	}
	if err == nil {
		t.Errorf("error should not be nil")
	} else {
		if !strings.Contains(err.Error(), "no data was loaded") {
			t.Errorf("expected error: 'no data was loaded', got: '%v'", err)
		}
	}

}
