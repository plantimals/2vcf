BINARY_PATH=bin
2VCF_BINARY=2vcf
2VCF_PATH=./

build: $(2VCF_BINARY)

$(2VCF_BINARY):
	go build -o ${BINARY_PATH}/${2VCF_BINARY} ${2VCF_PATH}

build2:
	go build -o ${BINARY} cli/2vcf.go

clean:
	rm -rf bin && go clean

windows:
	GOOS=windows GOARCH=amd64 go build -o 2vcf.exe 2vcf.go

linux:
	GOOS=linux GOARCH=amd64 go build -o 2vcf 2vcf.go
