BINARY=2vcf

build:
	go build -o ${BINARY}

clean:
	go clean


windows:
	GOOS=windows GOARCH=386 go build -o 2vcf.exe 2vcf.go
