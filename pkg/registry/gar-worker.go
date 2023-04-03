package registry

import (
	"context"
	"fmt"
	"strings"
	"time"

	artifactregistry "cloud.google.com/go/artifactregistry/apiv1"
	"cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
	"github.com/digtux/laminar/pkg/cfg"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

// getRegistries will attempt to find google artifact registries from an example docker image string
// For example:
// europe-docker.pkg.dev/your-project-id/your-registry-name
func getRegistries(ctx context.Context, log *zap.SugaredLogger, client artifactregistry.Client, registry cfg.DockerRegistry) ([]string, error) {
	searchString := registry.Reg
	domain := strings.Split(searchString, "/")[0]                 // would extract europe-docker.pkg.dev
	location := strings.ReplaceAll(domain, "-docker.pkg.dev", "") // would extract "europe"
	projectID := strings.Split(searchString, "/")[1]              // would extract "your-project-id"
	repoList, err := listGoogleArtifactRepositories(ctx, log, client, projectID, location)
	if err != nil {
		log.Errorw("couldn't list repositories",
			"error", err,
			"searchString", searchString,
		)
	}
	if len(repoList) < 1 {
		log.Warnw("scanning for Google Artifact Repositories didn't match anything",
			"registry", registry.Reg,
			"hint", "Expected repo name such as: 'europe-docker.pkg.dev/your-project-id/your-registry-name'",
		)
		return repoList, err
	}
	log.Debugw("Found Google Artifact Repositories",
		"repoList", repoList,
	)
	return repoList, err
}

func GarWorker(db *buntdb.DB, registry cfg.DockerRegistry, imageList []string, log *zap.SugaredLogger) {
	timeStart := time.Now()
	totalTags := 0

	ctx := context.Background()
	client := newClient(ctx, log)
	// defer client.Close but also check for errors
	defer func(client *artifactregistry.Client) {
		err := client.Close()
		if err != nil {
			log.Fatalw("unexpected error closing google artifact registry client",
				"error", err)
		}
	}(client)

	garRepos, err := getRegistries(ctx, log, *client, registry)
	if err != nil {
		log.Fatalw("couldn't get registries",
			"err", err,
		)
	}

	for _, repo := range garRepos {
		total := garDescribeAllRepositoryImagesToCache(ctx, log, *client, repo, db)
		totalTags += total
	}

	elapsed := time.Since(timeStart)
	log.Infow("Google Artifact Registry scan complete",
		"elapsed", elapsed,
		"registry", registry.Reg,
		"totalUniqueImages", len(imageList),
		"totalTags", totalTags,
	)
}

func newClient(ctx context.Context, log *zap.SugaredLogger) *artifactregistry.Client {
	client, err := artifactregistry.NewClient(ctx)
	if err != nil {
		log.Fatalw("Couldn't auth.. did you run: 'gcloud auth application-default login' ?",
			"error", err,
		)
	}
	return client
}

func listGoogleArtifactRepositories(
	ctx context.Context, log *zap.SugaredLogger, client artifactregistry.Client,
	projectID,
	location string,
) (
	[]string,
	error,
) {
	var result []string
	var err error

	// parent formatting is important
	// See: https://cloud.google.com/artifact-registry/docs/reference/rest/v1/projects.locations.repositories/list
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)
	it := client.ListRepositories(ctx, &artifactregistrypb.ListRepositoriesRequest{
		Parent: parent,
	})

	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return result, err
		}
		if resp.Format.String() == "DOCKER" {
			log.Debugw("Google artifact registry (format: DOCKER) found",
				"name", resp.Name,
				"format", resp.Format,
				"description", resp.Description,
			)
			result = append(result, resp.Name)
		}
	}
	return result, err
}

// TODO: figure out how to scan an individual dockerImage
func garDescribeAllRepositoryImagesToCache(
	ctx context.Context, log *zap.SugaredLogger, client artifactregistry.Client,
	repository string,
	db *buntdb.DB,
) (totalTags int) { // parent := repository,
	// "projects/<projectID>/locations/<location>/repositories/<repoName>"
	it := client.ListDockerImages(ctx, &artifactregistrypb.ListDockerImagesRequest{
		Parent: repository,
	})

	countUniqueTags := 0

	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Panic(err)
		}
		// TODO: we assume there are tags on an image.
		// this might be complicated for some folks might use raw sha256
		for _, tag := range resp.Tags {
			tagInfo := convertGarResponseToTagInfo(log, resp, tag)
			TagInfoToCache(tagInfo, db, log)
			countUniqueTags++
		}
	}
	log.Infow("Google Artifact Registry scanned",
		"countUniqueTags", countUniqueTags,
	)
	return countUniqueTags
}

func convertGarResponseToTagInfo(log *zap.SugaredLogger, resp *artifactregistrypb.DockerImage, tag string) TagInfo {
	// formats DockerImage data
	//
	// Uri example: "europe-docker.pkg.dev/acme-org/my-registry/image-name@sha256:8a1aa5d3eeee07bf5cd75cd1268e132a880ffc829dd02b059b6e68563219522b
	//
	// imageName: "europe-docker.pkg.dev/acme-org/my-registry/image-name"
	imageName := strings.Split(resp.Uri, "@")[0]
	// splitHash: "8a1aa5d3eeee07bf5cd75cd1268e132a880ffc829dd02b059b6e68563219522b"
	splitHash := strings.Split(resp.Uri, ":")
	if len(splitHash) != 2 {
		log.Fatalw("Expected just a single ':'",
			"value", resp.Uri,
			"splitHash", splitHash,
			"count", len(splitHash),
		)
	}
	hash := splitHash[1]

	// the .BuildTime contains two numbers
	// - the unix epoch in second
	// - the nanosecond underneath that second
	// This converts them to a regular time.Time
	created := time.Unix(
		resp.BuildTime.GetSeconds(),
		int64(resp.BuildTime.GetNanos()),
	)
	imageData := TagInfo{
		Image:   imageName,
		Tag:     tag,
		Hash:    hash,
		Created: created,
	}
	return imageData
}
