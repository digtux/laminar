package cache

import (
	"github.com/digtux/laminar/pkg/logger"
	"github.com/tidwall/buntdb"
)

// Open instantiates a lightweight buntdb "cache"
func Open(cacheLocation string) (db *buntdb.DB) {
	logger.Debugw("Cache initialised",
		"cacheLocation", cacheLocation,
	)
	db, err := buntdb.Open(cacheLocation)
	if err != nil {
		logger.Error(err)
	}
	// CreateIndexJson(db, "tag", "TagInfo:*", "tag")
	// CreateIndexJson(db, "image", "TagInfo:*", "image")
	// CreateIndexJson(db, "created", "TagInfo:*", "created")

	err = db.CreateIndex("hash", "TagInfo:*", buntdb.IndexJSON("image"))
	if err != nil {
		logger.Error(err)
	}
	err = db.CreateIndex("created", "TagInfo:*", buntdb.IndexJSON("created"))
	if err != nil {
		logger.Error(err)
	}
	err = db.CreateIndex("tag", "TagInfo:*", buntdb.IndexJSON("tag"))
	if err != nil {
		logger.Error(err)
	}
	return db
}
