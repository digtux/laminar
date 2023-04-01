package registry

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digtux/laminar/pkg/cfg"
	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
)

func GcrWorker(db *buntdb.DB, registry cfg.DockerRegistry, imageList []string, log *zap.SugaredLogger) {
	log.Fatalw("Sorry, GCR is not supported yet..")
	// TODO: GCR support
	root := registry.Reg
	ctx := context.Background()
	err := ls(ctx, root, true, false)
	log.Fatalw("Sorry, no GCR support yet",
		"error", err,
	)
}

func ls(ctx context.Context, root string, recursive, j bool) error {
	// working example (Apache2):
	// https://github.com/google/go-containerregistry/blob/1cfe1fc25f233b40aa5d3b0edd572ed5c3f854c9/cmd/gcrane/cmd/list.go#L57-L94
	repo, err := name.NewRepository(root)
	if err != nil {
		return err
	}

	opts := []google.Option{
		google.WithAuthFromKeychain(gcrane.Keychain),
		google.WithUserAgent("laminar"),
		google.WithContext(ctx),
	}

	if recursive {
		return google.Walk(repo, printImages(j), opts...)
	}

	tags, err := google.List(repo, opts...)
	if err != nil {
		return err
	}

	if !j {
		if len(tags.Manifests) == 0 && len(tags.Children) == 0 {
			// If we didn't see any GCR extensions, just list the tags like normal.
			for _, tag := range tags.Tags {
				fmt.Printf("%s:%s\n", repo, tag)
			}
			return nil
		}

		// Since we're not recursing, print the subdirectories too.
		for _, child := range tags.Children {
			fmt.Printf("%s/%s\n", repo, child)
		}
	}

	return printImages(j)(repo, tags, err)
}

func printImages(j bool) google.WalkFunc {
	return func(repo name.Repository, tags *google.Tags, err error) error {
		if err != nil {
			return err
		}

		if j {
			b, err := json.Marshal(tags)
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", b)
			return nil
		}

		for digest, manifest := range tags.Manifests {
			fmt.Printf("%s@%s\n", repo, digest)

			for _, tag := range manifest.Tags {
				fmt.Printf("%s:%s\n", repo, tag)
			}
		}

		return nil
	}
}
