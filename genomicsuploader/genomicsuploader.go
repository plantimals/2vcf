package genomicsuploader

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/briandowns/spinner"

	"cloud.google.com/go/storage"

	"golang.org/x/oauth2/google"

	"path"

	"encoding/json"

	"google.golang.org/api/genomics/v1"
)

//Client serves as the base for the the uploader
type Client struct {
	Project         string
	DatasetName     string
	VariantSetName  string
	StagingBucket   string
	GenomicsService *genomics.Service
	GCSClient       *storage.Client
}

type importResponse struct {
	Name string `json:"name"`
}

var validURL = regexp.MustCompile(`^gs://.*$`)

//New returns a genomics client for importing VCFs to google genomics
func New(project string, stagingBucket string) (*Client, error) {
	oauthHTTPClient, err := google.DefaultClient(context.Background(), genomics.GenomicsScope)
	if err != nil {
		return nil, err
	}
	ggService, err := genomics.New(oauthHTTPClient)
	if err != nil {
		return nil, err
	}
	GCSClient, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, err
	}
	return &Client{
		Project:         project,
		GenomicsService: ggService,
		GCSClient:       GCSClient,
		DatasetName:     "my data",
		VariantSetName:  "my variants",
	}, nil
}

//ImportVCF takes a path to your VCF and a staging bucket URL
func (c *Client) ImportVCF(inputFile string, bucket string) error {
	c.StagingBucket = bucket

	start := time.Now()
	gcsURL, err := c.stageVCF(inputFile, bucket)
	if err != nil {
		return err
	}
	log.Printf("finished staging in: %.2f seconds\n", time.Since(start).Seconds())

	log.Println("about to create dataset")
	ds, err := c.getOrCreateDataset()
	if err != nil {
		return err
	}

	log.Println("about to create variantset")
	vs, err := c.getOrCreateVariantset(ds)
	if err != nil {
		return err
	}

	ivr := c.GenomicsService.Variants.Import(&genomics.ImportVariantsRequest{
		Format:       "FORMAT_VCF",
		VariantSetId: vs.Id,
		SourceUris:   []string{gcsURL},
	})

	vr, err := ivr.Do()
	if err != nil {
		return err
	}

	vrBytes, err := vr.MarshalJSON()
	impResp := importResponse{}
	json.Unmarshal(vrBytes, &impResp)
	fmt.Printf("check on import progress by using the command: \n\tgcloud alpha genomics operations describe %s\n", impResp.Name)
	fmt.Print(string(vrBytes))
	return nil
}

func (c *Client) getOrCreateDataset() (*genomics.Dataset, error) {

	dslc := c.GenomicsService.Datasets.List().ProjectId(c.Project)
	dsResp, err := dslc.Do()
	if err != nil {
		return nil, err
	}
	log.Println("listed datasets")
	for _, dataset := range dsResp.Datasets {
		if dataset.Name == c.DatasetName {
			log.Printf("found existing dataset \"%s\"\n", c.DatasetName)
			return dataset, nil
		}
	}

	log.Printf("could not find Dataset named \"%s\", creating it now\n", c.DatasetName)

	dc := c.GenomicsService.Datasets.Create(&genomics.Dataset{
		Name:      c.DatasetName,
		ProjectId: c.Project,
	})
	ds, err := dc.Do()
	if err != nil {
		return nil, err
	}
	return ds, nil
}

func (c *Client) getOrCreateVariantset(ds *genomics.Dataset) (*genomics.VariantSet, error) {

	vslc := c.GenomicsService.Variantsets.Search(&genomics.SearchVariantSetsRequest{
		DatasetIds: []string{ds.Id},
	})
	vsResp, err := vslc.Do()
	if err != nil {
		return nil, err
	}

	for _, vs := range vsResp.VariantSets {
		if vs.Name == c.VariantSetName {
			log.Printf("found existing variantset \"%s\"\n", c.VariantSetName)
			return vs, nil
		}
	}

	vscc := c.GenomicsService.Variantsets.Create(&genomics.VariantSet{
		Name:      c.VariantSetName,
		DatasetId: ds.Id,
	})
	vs, err := vscc.Do()
	if err != nil {
		return nil, err
	}
	return vs, nil
}

func (c *Client) stageVCF(inputFile string, bucket string) (string, error) {
	staging := c.GCSClient.Bucket(bucket)
	filename := path.Base(inputFile)

	gcsURL := fmt.Sprintf("gs://%s/%s", bucket, filename)
	log.Println(gcsURL)

	obj := staging.Object(filename)

	if c.objectExists(bucket, filename) {
		return gcsURL, nil
	}

	upload := obj.NewWriter(context.Background())
	upload.ChunkSize = 10 * 256 * 1024
	defer upload.Close()

	file, err := os.Open(inputFile)
	if err != nil {
		return "", err
	}
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Prefix = "uploading vcf   "
	s.Start()

	b, err := io.Copy(upload, file)
	if err != nil {
		return "", err
	}
	s.Stop()
	log.Printf("copied %v bytes to staging bucket\n", b)
	return gcsURL, nil
}

func (c *Client) objectExists(bucket, object string) bool {
	it := c.GCSClient.Bucket(bucket).Objects(context.Background(), nil)
	for {
		itObj, err := it.Next()
		if err != nil {
			break
		}
		if itObj.Name == object {
			return true
		}
	}
	return false
}
