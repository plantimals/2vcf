/*Package convert handles the merging of microarray
genotype calls with the more detailed information
required for a VCF.

Included is a VCF with the union of the loci included
in 23andme and ancestry.com microarray chips. The
convert package intersects this VCF with the sites in
the raw data by marker name, the `rsid`.
*/
package convert
