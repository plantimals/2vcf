package genomicsuploader

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"

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
	Logger          *log.Logger
}

type importResponse struct {
	Name string `json:"name"`
}

var (
	logColor = color.New(color.FgRed).SprintFunc()
)

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
	logger := log.New(os.Stderr, logColor("[UPLOADER] "), log.Ldate|log.Ltime)
	return &Client{
		Project:         project,
		GenomicsService: ggService,
		GCSClient:       GCSClient,
		StagingBucket:   stagingBucket,
		DatasetName:     "my data",
		VariantSetName:  "my variants",
		Logger:          logger,
	}, nil
}

//ImportVCF takes a path to your VCF
func (c *Client) ImportVCF(inputFile string) error {

	start := time.Now()
	gcsURL, err := c.stageVCF(inputFile, c.StagingBucket)
	if err != nil {
		return err
	}
	if err := c.setAttributes(inputFile, c.StagingBucket); err != nil {
		return err
	}
	c.Logger.Printf("finished staging in: %.2f seconds\n", time.Since(start).Seconds())

	c.Logger.Println("about to create dataset")
	ds, err := c.getOrCreateDataset()
	if err != nil {
		return err
	}

	c.Logger.Println("about to create variantset")
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
	fmt.Printf("\n\ncheck on import progress by using the command: \n\tgcloud alpha genomics operations describe %s\n", impResp.Name)
	return nil
}

func (c *Client) getOrCreateDataset() (*genomics.Dataset, error) {

	dslc := c.GenomicsService.Datasets.List().ProjectId(c.Project)
	dsResp, err := dslc.Do()
	if err != nil {
		return nil, err
	}
	c.Logger.Println("listed datasets")
	for _, dataset := range dsResp.Datasets {
		if dataset.Name == c.DatasetName {
			c.Logger.Printf("found existing dataset \"%s\"\n", c.DatasetName)
			return dataset, nil
		}
	}

	c.Logger.Printf("could not find Dataset named \"%s\", creating it now\n", c.DatasetName)

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
			c.Logger.Printf("found existing variantset \"%s\"\n", c.VariantSetName)
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
	c.Logger.Println(gcsURL)

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
	s.Prefix = "uploading vcf can take serveral minutes  "
	s.Start()

	b, err := io.Copy(upload, file)
	if err != nil {
		return "", err
	}

	s.Stop()

	c.Logger.Printf("copied %v bytes to staging bucket\n", b)

	return gcsURL, nil
}

func (c *Client) setAttributes(inputFile, bucket string) error {
	filename := path.Base(inputFile)

	file := c.GCSClient.Bucket(bucket).Object(filename)

	_, err := file.Update(context.Background(), storage.ObjectAttrsToUpdate{
		ContentType:     "text/plain",
		ContentEncoding: "gzip",
	})
	if err != nil {
		return err
	}

	return nil
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
