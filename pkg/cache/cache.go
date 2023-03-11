package cache

import (
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
)

// Open instantiates a lightweight buntdb "cache"
func Open(cacheLocation string, log *zap.SugaredLogger) (db *buntdb.DB) {
	log.Debugw("Cache initialised",
		"cacheLocation", cacheLocation,
	)
	db, err := buntdb.Open(cacheLocation)
	if err != nil {
		log.Error(err)
	}
	// CreateIndexJson(db, "tag", "TagInfo:*", "tag")
	// CreateIndexJson(db, "image", "TagInfo:*", "image")
	// CreateIndexJson(db, "created", "TagInfo:*", "created")

	err = db.CreateIndex("hash", "TagInfo:*", buntdb.IndexJSON("image"))
	if err != nil {
		log.Error(err)
	}
	err = db.CreateIndex("created", "TagInfo:*", buntdb.IndexJSON("created"))
	if err != nil {
		log.Error(err)
	}
	err = db.CreateIndex("tag", "TagInfo:*", buntdb.IndexJSON("tag"))
	if err != nil {
		log.Error(err)
	}
	return db
}
