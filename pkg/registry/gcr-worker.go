package registry

import (
	"github.com/digtux/laminar/pkg/cfg"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
)

func GcrWorker(db *buntdb.DB, registry cfg.DockerRegistry, imageList []string, log *zap.SugaredLogger) {
	panic("GCR worker is not implemented")
}
