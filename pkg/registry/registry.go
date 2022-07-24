package registry

import (
	"encoding/json"
	"fmt"
	"github.com/digtux/laminar/pkg/cfg"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
	"strings"
	"time"
)

// TagInfo is the official data stored in buntDB
type TagInfo struct {
	Image   string    `json:"image"`
	Hash    string    `json:"hash"`
	Created time.Time `json:"created"`
	Tag     string    `json:"tag"`
}

type Client struct {
	db     *buntdb.DB
	logger *zap.SugaredLogger
}

func New(logger *zap.SugaredLogger, db *buntdb.DB) *Client {
	return &Client{
		logger: logger,
		db:     db,
	}
}

// Exec will check if we support that docker reg and then launch an appropriate worker
func (c *Client) Exec(registry cfg.DockerRegistry, imageList []string) {

	// grok will add some defaults lest the config doesn't include em
	registry = grokRegistrySettings(registry)
	c.logger.Debugw("DockerRegistry worker launching",
		"Registry", registry,
	)

	// Check if the image looks like an ECR image
	if strings.Contains(registry.Reg, "ecr") {
		EcrWorker(c.db, registry, imageList, c.logger)
		return
	}

	if strings.Contains(registry.Reg, "gcr") {
		GcrWorker(c.db, registry, imageList, c.logger)
		return
	}

	c.logger.Fatal("unable to figure out which kind of registry you have")
}

// incase some fields are missing, lets set their defaults
func grokRegistrySettings(in cfg.DockerRegistry) cfg.DockerRegistry {

	// incase this is empty set it..
	if in.TimeOut == 0 {
		in.TimeOut = 30
	}
	return in
}

func (c *Client) CachedImagesToTagInfoListSpecificImage(
	imageString string,
	index string,
) (result []TagInfo) {
	c.db.View(func(tx *buntdb.Tx) error {
		tx.Descend(index, func(key, val string) bool {
			// decode the data from the db
			x := JsonStringToTagInfo(val, c.logger)

			// if this image matches the imageString append it to the result
			if x.Image == imageString {
				result = append(result, x)
			}
			return true
		})
		return nil
	})
	//log.Debugw("searched DB for images",
	//	"image", imageString,
	//	"hits", len(result),
	//)
	return result
}

func JsonStringToTagInfo(s string, log *zap.SugaredLogger) TagInfo {
	var data TagInfo
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		log.Error("unmarshal error?")
		log.Fatal(err)
		return data
	}
	return data
}

func TagInfoToCache(info TagInfo, db *buntdb.DB, log *zap.SugaredLogger) {
	StorageKey := fmt.Sprintf("TagInfo:%s:%s:%s", info.Image, info.Hash, info.Tag)

	// TTL on tag cache, https://github.com/tidwall/buntdb#data-expiration
	Opts := &buntdb.SetOptions{Expires: true, TTL: time.Second * 300}

	byteArray, err := json.Marshal(info)
	if err != nil {
		log.Fatal(err)
	}

	insertedValue := string(byteArray)
	//log.Debugf("writing to cache: key: '%s', value: '%s'", StorageKey, insertedValue)
	err = db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(StorageKey, insertedValue, Opts)
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
}
