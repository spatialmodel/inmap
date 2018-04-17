FROM golang

WORKDIR /go/src/github.com/spatialmodel/inmap

RUN git init
RUN git remote add origin https://github.com/spatialmodel/inmap.git
RUN git fetch origin emis
RUN git pull origin emis --depth 1

RUN go get github.com/golang/dep/cmd/dep
RUN dep ensure

WORKDIR /go/src/github.com/spatialmodel/inmap/cmd/inmapweb

RUN go install

# This step is so we can find the test certificates.
RUN go get google.golang.org/grpc/testdata

ENV SLCASpatialCache https://storage.googleapis.com/inmapweb/spatialeio

ENTRYPOINT /go/bin/inmapweb

EXPOSE 10000
