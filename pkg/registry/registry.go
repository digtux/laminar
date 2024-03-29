package registry

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/logger"
	"github.com/tidwall/buntdb"
)

// TagInfo is the official data stored in buntDB
type TagInfo struct {
	Image   string    `json:"image"`
	Hash    string    `json:"hash"`
	Created time.Time `json:"created"`
	Tag     string    `json:"tag"`
}

type Client struct {
	db *buntdb.DB
}

func New(db *buntdb.DB) *Client {
	return &Client{
		db: db,
	}
}

// Exec will check if we support that docker reg and then launch an appropriate worker
func (c *Client) Exec(registry cfg.DockerRegistry, imageList []string) { // grok will add some defaults lest the config doesn't include em
	registry = grokRegistrySettings(registry)
	logger.Debugw("DockerRegistry worker launching",
		"Registry", registry,
	)

	// Check if the image looks like an ECR image
	if strings.Contains(registry.Reg, "ecr") {
		EcrWorker(c.db, registry, imageList)
		return
	}

	if strings.Contains(registry.Reg, "gcr.io") {
		GcrWorker(c.db, registry, imageList)
		return
	}

	if strings.Contains(registry.Reg, "docker.pkg.dev") {
		GarWorker(c.db, registry, imageList)
		return
	}

	logger.Fatal("unable to figure out which kind of registry you have")
}

// assuming these are unset fields, assume these defaults
func grokRegistrySettings(in cfg.DockerRegistry) cfg.DockerRegistry {
	if in.TimeOut == 0 {
		in.TimeOut = 30
	}
	return in
}

func (c *Client) CachedImagesToTagInfoListSpecificImage(
	imageString string,
	index string,
) (result []TagInfo) {
	err := c.db.View(func(tx *buntdb.Tx) error {
		err := tx.Descend(index, func(key, val string) bool {
			// decode the data from the db
			x := JSONStringToTagInfo(val)

			// if this image matches the imageString append it to the result
			if x.Image == imageString {
				result = append(result, x)
			}
			return true
		})
		if err != nil {
			logger.Debugw("buntdb tx.Descend issue",
				"err", err)
			return err
		}
		return nil
	})
	if err != nil {
		logger.Debugw("buntdb tx.View() issue",
			"error", err)
		return nil
	}
	// log.Debugw("searched DB for images",
	//	"image", imageString,
	//	"hits", len(result),
	// )
	return result
}

func JSONStringToTagInfo(s string) TagInfo {
	var data TagInfo
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		logger.Error("unmarshal error?")
		logger.Fatal(err)
		return data
	}
	return data
}

func TagInfoToCache(info TagInfo, db *buntdb.DB) {
	storeKey := fmt.Sprintf("TagInfo:%s:%s:%s", info.Image, info.Hash, info.Tag)

	// TTL on tag cache, https://github.com/tidwall/buntdb#data-expiration
	buntOpts := &buntdb.SetOptions{Expires: true, TTL: time.Second * 300}

	byteArray, err := json.Marshal(info)
	if err != nil {
		logger.Fatal(err)
	}

	insertedValue := string(byteArray)
	err = db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(storeKey, insertedValue, buntOpts)
		return err
	})
	if err != nil {
		logger.Fatal(err)
	}
}
