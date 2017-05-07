# 2vcf - convert raw 23andme or ancestry.com data to VCF

The [VCF](https://samtools.github.io/hts-specs/VCFv4.3.pdf) is a widely adopted format for storing detailed data about genetic variation. Services like [23andme](https://www.23andme.com/) and [ancestry.com](https://www.ancestry.com/) offer to genotype customers at less than a million well-characterized sites in the human genome. It is possible to obtain the raw data collected by these sites, but the raw data are provided in a minimal format, which is not trivial to enrich and transform into the VCF format. _2vcf_ converts the raw output from 23andme or ancestry.com into a gzipped VCF file. The output vcf is populated with [human variant data](https://www.ncbi.nlm.nih.gov/variation/docs/human_variation_vcf/), which includes all alternate alleles, annotations, etc.

In order to build 2vcf, the [golang](https://golang.org/) build tool is required. On os x use [homebrew](https://brew.sh/) to install it `brew install go`. 

Build 2vcf by checking out the [source repo](https://github.com/plantimals/2vcf), entering the directory `cd 2vcf`, and running the make file `make`. Build for windows by using `make windows`.

Convert your raw data by running the utility `./2vcf --input-file my-raw-data.zip --output-file my-personal-genotypes.vcf.gz`. Running the utility from another location works as well, but remember to specify the path to the reference data as well `--vcf-ref /home/me/git/2vcf/reference.vcf.gz`.

Please report any errors or difficulties with the utility. 

