FROM golang

WORKDIR /go/src/github.com/spatialmodel

RUN git clone --depth=1 https://github.com/spatialmodel/inmap.git

WORKDIR /go/src/github.com/spatialmodel/inmap

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN dep ensure

RUN rm -r vendor/k8s.io
RUN go get k8s.io/client-go/...

RUN go install github.com/spatialmodel/inmap/cmd/...

# This step installs ssl certificates for testing.
RUN go get google.golang.org/grpc/testdata

ENV INMAP_ROOT_DIR /go/src/github.com/spatialmodel/inmap/

EXPOSE 10000
EXPOSE 8080
