/*2vcf is a converter for 23andme or ancestry genotype data into VCF format
 */
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/plantimals/2vcf/convert"
	"github.com/plantimals/2vcf/genomicsuploader"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	app = kingpin.New("2vcf", "convert raw genotype calls from sources like 23andme or ancestry.com into vcf format and upload it to google genomics")

	conv        = app.Command("conv", "convert raw data to vcf format")
	rawType     = conv.Arg("raw genotype source", "source of the raw genotype data, 23andme or ancestry.com").Required().Enum("23andme", "ancestry")
	inFile      = conv.Arg("input-data", "relative path to input data, zip or ascii").Required().ExistingFile()
	outFile     = conv.Flag("output-file", "relative path to output data, gzipped").Short('o').String()
	vcfRef      = conv.Flag("vcf-ref", "relative path to vcf reference data, gzipped").Default("reference/reference.vcf.gz").Short('v').String()
	convProject = conv.Flag("google-project", "the google cloud project to push your vcf into").Short('g').String()
	convBucket  = conv.Flag("bucket", "google cloud storage url to bucket to be used for staging").Short('b').String()
	convPush    = conv.Flag("push", "push generated VCF file into google genomics").Short('p').Bool()

	push        = app.Command("push", "push a vcf into google genomics")
	inFilePush  = push.Arg("input-data", "relative path to input data, zip or ascii").Required().ExistingFile()
	pushProject = push.Flag("google-project", "the google cloud project to push your vcf into").Required().Short('g').String()
	pushBucket  = push.Flag("bucket", "google cloud storage url to bucket to be used for staging").Required().Short('b').String()
	dsName      = push.Flag("dataset-name", "name for the dataset to load variants into. if it matches and existing dataset, 2vcf "+
		"will load your variants into the existing dataset, if it doesn't exist, "+
		"2vcf will create a dataset by that name").Default("2vcf dataset").Short('d').String()
	vsName = push.Flag("variantset-name", "name for your variants, will get or create a variantset of this name").Default("2vcf variants").Short('v').String()
)

var (
	cyan = color.New(color.FgCyan).SprintFunc()
)

func main() {
	app.UsageTemplate(kingpin.CompactUsageTemplate).Version("1.0.0").Author("Rob Long")
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case conv.FullCommand():
		RunConv()
	case push.FullCommand():
		RunPush(*inFilePush, *pushProject, *pushBucket, *dsName, *vsName)
	}
}

func RunConv() {
	outputFile := *outFile
	if outputFile == "" {
		outputFile = (*inFile)[:len(*inFile)-4] + ".vcf.gz"
	}
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Prefix = "converting raw data to vcf   "
	s.Start()

	client := convert.NewclientWithRef(*inFile, outputFile, *vcfRef)

	switch *rawType {
	case "23andme":
		client.Convert23AndMe()
	case "ancestry":
		client.ConvertAncestry()
	}
	s.Stop()

	fmt.Printf("\nvcf output at: %s\n\n", cyan(outputFile))

	if *convPush {
		if *convProject == "" || *convBucket == "" {
			kingpin.FatalUsage("if --push is used to push the output of conversion into google genomics, a project and bucket must be specified")
		}
		RunPush(outputFile, *convProject, *convBucket, "2vcf data", "2vcf variants")
	}
}

func RunPush(input string, project string, bucket string, datasetName string, variantsetName string) {

	ggClient, err := genomicsuploader.New(project, bucket)
	if err != nil {
		log.Fatal(err)
	}

	if err := ggClient.ImportVCF(input); err != nil {
		log.Fatal(err)
	}
}
