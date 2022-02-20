module github.com/spatialmodel/inmap

require (
	cloud.google.com/go/storage v1.15.0
	github.com/BurntSushi/toml v0.3.1
	github.com/GaryBoone/GoStats v0.0.0-20130122001700-1993eafbef57
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/aws/aws-sdk-go v1.38.35
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cenkalti/backoff/v4 v4.1.1
	github.com/cpuguy83/go-md2man v1.0.9-0.20180619205630-691ee98543af // indirect
	github.com/ctessum/atmos v0.0.0-20170526022537-cba69f7ca647
	github.com/ctessum/cdf v0.0.0-20181201011353-edced208ea9d
	github.com/ctessum/geom v0.2.10
	github.com/ctessum/go-leaflet v0.0.0-20170724133759-2f9e4c38fb5e
	github.com/ctessum/gobra v0.0.0-20180516235632-ddfa5eeb3017
	github.com/ctessum/plotextra v0.0.0-20180623195436-96488e3f1996
	github.com/ctessum/requestcache v1.0.1
	github.com/ctessum/requestcache/v2 v2.0.0
	github.com/ctessum/requestcache/v4 v4.0.0
	github.com/ctessum/sparse v0.0.0-20181201011727-57d6234a2c9d
	github.com/ctessum/unit v0.0.0-20160621200450-755774ac2fcb
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/distribution v2.8.0+incompatible // indirect
	github.com/go-humble/detect v0.1.2 // indirect
	github.com/go-humble/router v0.5.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/golang/protobuf v1.5.2
	github.com/gonum/floats v0.0.0-20181209220543-c233463c7e82
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e
	github.com/gopherjs/vecty v0.0.0-20180525005238-a3bd138280bf
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/hcl v0.0.0-20171017181929-23c074d0eceb // indirect
	github.com/improbable-eng/grpc-web v0.0.0-20190113155728-0c7a81a25d11
	github.com/jackc/pgx/v4 v4.12.0
	github.com/johanbrandhorst/protobuf v0.6.1
	github.com/jonas-p/go-shp v0.1.2-0.20190401125246-9fd306ae10a6
	github.com/kr/pretty v0.3.0
	github.com/lib/pq v1.10.2
	github.com/lnashier/viper v0.0.0-20180730210402-cc7336125d12
	github.com/magiconair/properties v1.7.3 // indirect
	github.com/mattn/go-sqlite3 v1.11.0
	github.com/moby/sys/mountinfo v0.6.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/opencontainers/runc v1.1.0 // indirect
	github.com/pelletier/go-toml v1.0.1 // indirect
	github.com/rs/cors v1.3.0 // indirect
	github.com/russross/blackfriday v2.0.0+incompatible // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/skratchdot/open-golang v0.0.0-20160302144031-75fb7ed4208c
	github.com/spf13/cast v1.2.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/jwalterweatherman v0.0.0-20170901151539-12bd96e66386 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/tealeg/xlsx v1.0.3
	github.com/testcontainers/testcontainers-go v0.11.1
	gocloud.dev v0.23.0
	golang.org/x/build v0.0.0-20190226180436-80ca8d25ddd4
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd
	golang.org/x/sys v0.0.0-20220209214540-3681064d5158 // indirect
	gonum.org/v1/gonum v0.0.0-20191009222026-5d5638e6749a
	gonum.org/v1/plot v0.0.0-20190526055220-ccfad0c86201
	google.golang.org/grpc v1.37.0
	honnef.co/go/js/dom v0.0.0-20180323154144-6da835bec70f
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v0.20.1
)

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999

go 1.13
