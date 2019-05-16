package convert

import (
	"archive/zip"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/biogo/hts/bgzf"
	"github.com/brentp/vcfgo"
	"gopkg.in/h2non/filetype.v1"
)

type Client struct {
	inputFile  string
	outputFile string
	vcfRef     string
	ancestry   bool
}

var (
	defaultVCFRef = "reference/reference.vcf.gz"
)

//Newclient returns a basic Client, with default reference path
func Newclient(inputFile string, outputFile string) *Client {
	return NewclientWithRef(inputFile, outputFile, "reference/reference.vcf.go")
}

//NewclientWithRef returns a new Client, containing the specified ref
func NewclientWithRef(inputFile string, outputFile string, vcfRef string) *Client {
	return &Client{
		inputFile:  inputFile,
		outputFile: outputFile,
		vcfRef:     vcfRef,
	}
}

type rsid string
type chrom string
type locus struct {
	chrom chrom  "chromosome name"
	rsid  rsid   "rsid of the marker"
	pos   int    "one based physical coordinate"
	gt    string "genotype call"
}

func (l locus) String() string {
	return fmt.Sprintf("%s\t%v\t%s", l.chrom, l.pos, l.rsid)
}

func errHndlr(msg string, err error) {
	if err != nil {
		fmt.Println(msg, err)
		os.Exit(1)
	}
}

func (c *Client) Convert23AndMe() {
	c.ancestry = false
	c.convertCalls()
}

func (c *Client) ConvertAncestry() {
	c.ancestry = true
	c.convertCalls()
}

func (c *Client) convertCalls() {
	loci := c.getLoci(c.inputFile)

	refRdr, ref := getRefReader(c.vcfRef)
	defer ref.Close()

	hdr := c.getHeader(refRdr)

	vcfOut, err := os.Create(c.outputFile)
	errHndlr("error opening file for vcf output: ", err)
	defer vcfOut.Close()

	bgzfOut := bgzf.NewWriter(vcfOut, gzip.BestCompression)
	defer bgzfOut.Flush()

	vcfWriter, err := vcfgo.NewWriter(bgzfOut, hdr)
	errHndlr("error opening vfgo.Writer: ", err)

	for {
		variant := refRdr.Read()
		if variant == nil {
			break
		}
		locus, ok := loci[rsid(variant.Id_)]
		if !ok {
			continue
		}
		variant.Samples = addGenotypeSample(locus, variant)
		vcfWriter.WriteVariant(variant)
	}
}

func (c *Client) getLoci(inputFile string) map[rsid]locus {
	var (
		input io.Reader
		err   error
	)

	if isZip(inputFile) {
		//log.Println("processing compressed input")
		zipIn, err := zip.OpenReader(inputFile)
		defer zipIn.Close()

		zipFile := zipIn.File[0]
		input, err = zipFile.Open()
		errHndlr("failed to read zipped input: ", err)
	} else {
		//log.Println("processing uncompressed input")
		input, err = os.Open(inputFile)
		errHndlr("failed to uncompressed input", err)
	}

	var inputScanner = bufio.NewScanner(input)

	loci := make(map[rsid]locus)
	for inputScanner.Scan() {
		line := inputScanner.Text()
		if line[0] == '#' || line[0:4] == "rsid" {
			continue
		}
		locus := c.parse(line)
		loci[locus.rsid] = locus
	}
	return loci
}

func isZip(inputFile string) bool {
	input, err := os.Open(inputFile)
	errHndlr("error checking input file for type", err)
	defer input.Close()
	bb := make([]byte, 100)
	input.Read(bb)
	kind, err := filetype.Match(bb)
	return kind.Extension == "zip"
}

func (c *Client) parse(line string) locus {
	s := strings.Split(line, "\t")
	pos, err := strconv.ParseInt(s[2], 10, 32)
	errHndlr("error parsing pos: "+s[2], err)

	// if reading ancestry data, alleles are split across
	// columns 3 and 4, rather than 23andme's joined alleles
	// in column 3
	alleles := s[3]
	if c.ancestry {
		alleles = s[3] + s[4]
	}

	return locus{chrom(s[1]), rsid(s[0]), int(pos), alleles}
}

func getRefReader(referenceFile string) (*vcfgo.Reader, *os.File) {
	ref, err := os.Open(referenceFile)
	errHndlr("error opening reference file: ", err)
	//defer ref.Close()
	refUnzip, err := gzip.NewReader(ref)
	errHndlr("error unzipping reference: ", err)

	refRdr, err := vcfgo.NewReader(refUnzip, true)
	errHndlr("error creating vcfgo.Reader: ", err)

	return refRdr, ref
}

func (c *Client) getHeader(rdr *vcfgo.Reader) *vcfgo.Header {
	hdr := rdr.Header
	hdr.SampleNames = []string{c.inputFileName()}
	hdr.SampleFormats["GT"] = getSampleFormatGT()
	return hdr
}

func getSampleFormatGT() *vcfgo.SampleFormat {
	var answer vcfgo.SampleFormat
	answer.Id = "GT"
	answer.Description = "Genotype"
	answer.Number = "."
	answer.Type = "Integer"
	return &answer
}

func addGenotypeSample(loc locus, variant *vcfgo.Variant) []*vcfgo.SampleGenotype {
	variant.Format = []string{"GT"}
	gt := vcfgo.NewSampleGenotype()
	gt.Phased = false
	gt.GT = loc.getGenotypeInts(variant)
	gt.Fields["GT"] = getGenotypeString(gt.GT)
	return []*vcfgo.SampleGenotype{gt}
}

func (l *locus) getGenotypeInts(v *vcfgo.Variant) []int {
	gts := append([]string{v.Reference}, v.Alternate...)
	answer := make([]int, len(l.gt))
	for lidx := 0; lidx < len(l.gt); lidx++ {
		for i, allele := range gts {
			if string(l.gt[lidx]) == allele {
				answer[lidx] = i
			}
		}
	}
	return answer
}

func getGenotypeString(gts []int) string {
	alleles := make([]string, len(gts))
	for i, gt := range gts {
		alleles[i] = strconv.Itoa(gt)
	}
	return strings.Join(alleles, "/")
}

func (c *Client) inputFileName() string {
	parts := strings.Split(filepath.Base(c.inputFile), ".")
	return strings.Join(parts[0:len(parts)-1], ".")
}
