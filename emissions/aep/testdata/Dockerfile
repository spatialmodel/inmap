
FROM postgis/postgis:14-3.2-alpine

# Install OSM2PGSQL
ENV OSM2PGSQL_VERSION 1.5.0

RUN apk add --no-cache cmake make g++ boost-dev expat-dev \
    bzip2-dev zlib-dev libpq proj-dev lua5.3-dev postgresql-dev git \ 
    &&\
	cd $HOME &&\
	mkdir src &&\
	cd src &&\
	git clone --depth 1 --branch $OSM2PGSQL_VERSION https://github.com/openstreetmap/osm2pgsql.git &&\
	cd osm2pgsql &&\
	mkdir build &&\
	cd build &&\
	cmake -DLUA_LIBRARY=/usr/lib/liblua-5.3.so.0 .. &&\
	make &&\
	make install

COPY honolulu_hawaii.osm.pbf /
