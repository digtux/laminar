package registry

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/digtux/laminar/pkg/cfg"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
)

// type SortImageIds []*ecr.ImageIdentifier
//
// func (c SortImageIds) Len() int      { return len(c) }
// func (c SortImageIds) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
// func (c SortImageIds) Less(i, j int) bool {
// 	// fmt.Println(*c[i].ImageTag, *c[j].ImageTag)
// 	if c[i].ImageTag == nil {
// 		return true
// 	}
// 	if c[j].ImageTag == nil {
// 		return false
// 	}
// 	return strings.Compare(*c[i].ImageTag, *c[j].ImageTag) == -1
// }

func EcrGetAuth(registry cfg.DockerRegistry) (svc *ecr.ECR) {
	mySession := session.Must(session.NewSession())
	myRegion := strings.Split(registry.Reg, ".")[3]
	svc = ecr.New(mySession, aws.NewConfig().WithRegion(myRegion))
	return svc
}

func EcrWorker(db *buntdb.DB, registry cfg.DockerRegistry, imageList []string, log *zap.SugaredLogger) {
	timeStart := time.Now()
	totalTags := 0
	auth := EcrGetAuth(registry)

	// EG "112233445566.dkr.ecr.eu-west-2.amazonaws.com/acmecorp/app-name" == "acmecorp"
	repoName := strings.Split(registry.Reg, "/")[1]

	for _, img := range imageList {
		log.Debugw("EcrWorker",
			"action", "scanning for image tags",
			"image", img,
		)

		// EG "112233445566.dkr.ecr.eu-west-2.amazonaws.com/acmecorp/app-name" == "app-name"
		imgName := strings.Split(img, "/")[2]

		name := fmt.Sprintf("%s/%s", repoName, imgName)
		newTagsCount := EcrDescribeImageToCache(auth, name, registry, db, log)
		totalTags += newTagsCount
	}

	elapsed := time.Since(timeStart)
	log.Infow("Amazon ECR scan complete",
		"elapsed", elapsed,
		"registry", registry.Reg,
		"totalImages", len(imageList),
		"totalTags", totalTags,
	)
}

func EcrDescribeImageToCache(
	svc *ecr.ECR,
	repositoryName string,
	registry cfg.DockerRegistry,
	db *buntdb.DB,
	log *zap.SugaredLogger,
) (total int) {
	total = 0
	describeImageSettings := &ecr.DescribeImagesInput{
		// EG: 112233445566.dkr.ecr.eu-west-2.amazonaws.com/acmecorp
		RepositoryName: aws.String(repositoryName),
	}

	var imageDetails []*ecr.ImageDetail

	// page through all ECR images and add them to the imageDetails slice
	// https://github.com/terraform-providers/terraform-provider-aws/pull/8403/files/83d482992b6c42bea36d94f14b1da6616dc81ad1
	err := svc.DescribeImagesPages(describeImageSettings, func(page *ecr.DescribeImagesOutput, lastPage bool) bool {
		imageDetails = append(imageDetails, page.ImageDetails...)
		return true
	})

	if err != nil {
		log.Fatalw("ECR DescribeImages failed",
			"error", err,
			"suggest", "maybe set $AWS_PROFILE?")
	}

	// only the AWS sdk required the "prefix/myimage" part of the repositoryName, afterwards lets remove that
	repositoryName = strings.Split(repositoryName, "/")[1]

	for _, hit := range imageDetails {
		for _, tag := range hit.ImageTags {
			total++
			cleanerDigest := strings.Split(*hit.ImageDigest, ":")[1]
			fullImageName := fmt.Sprintf("%s/%s", registry.Reg, repositoryName)

			hitTagInfo := &TagInfo{
				Image:   fullImageName,
				Hash:    cleanerDigest,
				Tag:     *tag,
				Created: *hit.ImagePushedAt,
			}
			TagInfoToCache(*hitTagInfo, db, log)
		}
	}
	log.Debugw("indexing image complete",
		"registryUrl", registry.Reg,
		"registryName", registry.Name,
		"images", repositoryName,
		"totalTags", len(imageDetails),
	)
	return total
}
