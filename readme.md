## 2vcf 

in order to improve individual sovereignty over genetic/genomic information, facilitate a deeper understanding of biology and computation, and promote shared meaning, openb.io provides `2vcf` under the [MIT license](https://mit-license.org). `2vcf` will convert raw genotype data exports from [23andme](https://www.23andme.com) or [Ancestry.com](https://www.ancestry.com) into [VCF format](https://samtools.github.io/hts-specs/VCFv4.2.pdf).

`2vcf` produces a VCF that contains annotations from dbSNP [build 151](https://github.com/ncbi/dbsnp/tree/master/Build%20Announcements/151) on `GRCh37.p13`. these annotations include allele frequencies from various sources including [1000 Genomes](https://www.internationalgenome.org) and [ExAC](http://exac.broadinstitute.org/), [RefSeq](https://www.ncbi.nlm.nih.gov/refseq/) gene annotations, and functional class of the variant.

the source VCF for dbSNP build 151 weighs in at around 15GB. the sites assayed by personal genomics companies are but a tiny fraction of the totality of dbSNP sites. so I make available a reference version of the dbSNP VCF which has been filtered down to those sites likely to be contained in your exported 23andme or Ancestry.com exported raw data. for more details on which sites are included and why, see this writeup on the sources for `2vcf reference v2.0`.

## usage

1. download the appropriate binary for your architecture from the [most recent github release](https://github.com/plantimals/2vcf/releases/tag/v0.4.0). un-tar the contents after downloading.

2. download the [reference vcf](http://openb.io/2vcf/2vcf-v2.0.vcf.gz) http://openb.io/2vcf/2vcf-v2.0.vcf.gz

3. download your raw genotype data from [23andme](https://customercare.23andme.com/hc/en-us/articles/212196868-Accessing-and-Downloading-Your-Raw-Data) or [Ancestry](https://support.ancestry.com/s/article/Downloading-AncestryDNA-Raw-Data).

4. now run the `2vcf` binary with the appropriate options:

```
./2vcf conv 23andme --ref path/to/2vcf-v2.0.vcf.gz \
    --input path/to/my/raw/genotypes.zip \
    --output my-personal-annotated.vcf.gz
```

Please report any errors or difficulties with the utility. 

