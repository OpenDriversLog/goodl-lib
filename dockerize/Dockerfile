FROM golang:1.6

RUN apt-get update && apt-get install -y --no-install-recommends \
        golang-go.tools \
     && rm -rf /var/lib/apt/lists/*

RUN go get -v github.com/onsi/ginkgo/ginkgo \
    && go get -v github.com/onsi/gomega

RUN mkdir /databases
ENV ENVIRONMENT test
ENV GO15VENDOREXPERIMENT 1
RUN go get -v github.com/Masterminds/glide

COPY . $GOPATH/src/github.com/OpenDriversLog/goodl-lib

WORKDIR $GOPATH/src/github.com/OpenDriversLog/goodl-lib

RUN glide up

# quick fix for repo breaking the build:
#   vendor/github.com/denisenkom/go-mssqldb/examples/tsql.go:15: main redeclared in this block
#	previous declaration at vendor/github.com/denisenkom/go-mssqldb/examples/simple.go:15
RUN rm -r vendor/github.com/denisenkom/go-mssqldb/examples

# quick fix for repo breaking the build:
#go build github.com/OpenDriversLog/goodl-lib/vendor/google.golang.org/grpc/test: no buildable Go source files in /go/src/github.com/OpenDriversLog/goodl-lib/vendor/google.golang.org/grpc/test
#RUN rm -r vendor/google.golang.org/grpc/test

#RUN go build -v ./...

CMD ["/bin/bash"]
