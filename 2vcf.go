/*2vcf is a converter for 23andme or ancestry genotype data into VCF format
 */
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/plantimals/2vcf/convert"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	app = kingpin.New("2vcf", "convert raw genotype calls from sources like 23andme or ancestry.com into vcf format with dbSNP annotations")

	conv        = app.Command("conv", "convert raw data to vcf format")
	rawType     = conv.Arg("23andme or ancestry", "source of the raw genotype data, 23andme or ancestry.com").Required().Enum("23andme", "ancestry")
	inFile      = conv.Flag("input", "path to input data, zip or ascii").Required().Short('i').ExistingFile()
	vcfRef      = conv.Flag("ref", "path to gzipped vcf reference data").Required().Short('r').String()
	outFile     = conv.Flag("output", "path to output data, gzipped").Short('o').String()
	fixAllosome = conv.Flag("fixAllos", "double allosomal calls for downstream analysis").Short('a').Bool()
)

var (
	cyan = color.New(color.FgCyan).SprintFunc()
)

func main() {
	app.UsageTemplate(kingpin.CompactUsageTemplate).Version("1.0.0").Author("Rob Long")
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case conv.FullCommand():
		RunConv()
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

	client := convert.NewclientWithRef(*inFile, outputFile, *vcfRef, *fixAllosome)

	switch *rawType {
	case "23andme":
		client.Convert23AndMe()
	case "ancestry":
		client.ConvertAncestry()
	}
	s.Stop()

	fmt.Printf("\nvcf output at: %s\n\n", cyan(outputFile))
}
