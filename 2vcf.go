package main

import (
	"archive/zip"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"

	"github.com/biogo/hts/bgzf"
	"github.com/brentp/vcfgo"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/h2non/filetype.v1"
)

var (
	ancestry   = kingpin.Flag("ancestry", "input-data is from ancestry.com").Bool()
	inputFile  = kingpin.Flag("input-data", "relative path to input data, zip or ascii").Required().Short('i').String()
	outputFile = kingpin.Flag("output-data", "relative path to output data, gzipped").Required().Short('o').String()
	vcfRef     = kingpin.Flag("vcf-ref", "relative path to vcf reference data, gzipped").Default("reference.vcf.gz").Short('v').String()
	profiling  = false
)

type Rsid string
type Chrom string
type Locus struct {
	chrom Chrom  "chromosome name"
	rsid  Rsid   "rsid of the marker"
	pos   int    "one based physical coordinate"
	gt    string "genotype call"
}

func (l Locus) String() string {
	return fmt.Sprintf("%s\t%v\t%s", l.chrom, l.pos, l.rsid)
}

func errHndlr(msg string, err error) {
	if err != nil {
		fmt.Println(msg, err)
		os.Exit(1)
	}
}

func main() {
	if profiling {
		f, err := os.Create("profile.out")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Version("1.0.0")
	kingpin.Parse()

	convertCalls(*inputFile, *vcfRef, *outputFile)
}

func convertCalls(inputFile string, referenceFile string, outputFile string) {
	loci := getLoci(inputFile)

	refRdr, ref := getRefReader(referenceFile)
	defer ref.Close()

	hdr := getHeader(refRdr)

	vcfOut, err := os.Create(outputFile)
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
		locus, ok := loci[Rsid(variant.Id_)]
		if !ok {
			continue
		}
		variant.Samples = addGenotypeSample(locus, variant)
		vcfWriter.WriteVariant(variant)
	}
}

func getLoci(inputFile string) map[Rsid]Locus {
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

	loci := make(map[Rsid]Locus)
	for inputScanner.Scan() {
		line := inputScanner.Text()
		if line[0] == '#' || line[0:4] == "rsid" {
			continue
		}
		locus := parse(line)
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

func parse(line string) Locus {
	s := strings.Split(line, "\t")
	pos, err := strconv.ParseInt(s[2], 10, 32)
	errHndlr("error parsing pos: "+s[2], err)

	// if reading ancestry data, alleles are split across
	// columns 3 and 4, rather than 23andme's joined alleles
	// in column 3
	alleles := s[3]
	if *ancestry {
		alleles = s[3] + s[4]
	}

	return Locus{Chrom(s[1]), Rsid(s[0]), int(pos), alleles}
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

func getHeader(rdr *vcfgo.Reader) *vcfgo.Header {
	hdr := rdr.Header
	hdr.SampleNames = []string{inputFileName()}
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

func addGenotypeSample(locus Locus, variant *vcfgo.Variant) []*vcfgo.SampleGenotype {
	variant.Format = []string{"GT"}
	gt := vcfgo.NewSampleGenotype()
	gt.Phased = false
	gt.GT = locus.getGenotypeInts(variant)
	gt.Fields["GT"] = getGenotypeString(gt.GT)
	return []*vcfgo.SampleGenotype{gt}
}

func (l *Locus) getGenotypeInts(v *vcfgo.Variant) []int {
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

func inputFileName() string {
	parts := strings.Split(filepath.Base(*inputFile), ".")
	return strings.Join(parts[0:len(parts)-1], ".")
}
