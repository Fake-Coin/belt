all: bindata.go
	CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build fakco.in/belt/cmd/belt
	docker build -t neilvallon/belt .

deploy: all
	docker push neilvallon/belt
	
bindata.go: assets
	go-bindata-assetfs -pkg belt ./assets/...

clean:
	rm belt || true
	rm bindata.go || true