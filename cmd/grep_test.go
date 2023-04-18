package cmd

import (
	"testing"

	"github.com/digtux/laminar/pkg/logger"
)

func init() {
	logger.InitLogger(true)
}

func TestMatchSemver(t *testing.T) {
	semverTests := []struct {
		version    string
		constraint string
		output     bool
	}{
		{version: "v0.0.1", constraint: "thisisfake", output: false},
		{version: "thisisfake", constraint: ">=0.0.0", output: false},
		{version: "v0.0.1", constraint: ">=0.0.0", output: true},
		{version: "v0.0.1", constraint: ">=1.0.0", output: false},
	}

	for _, test := range semverTests {
		s := matchSemver(test.version, test.constraint)
		if s != test.output {
			t.Errorf("testSemver(%s, %s), got: '%t' but expected: '%t'", test.version, test.constraint, s, test.output)
		}
	}

}
