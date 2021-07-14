# Copyright (c) 2014, Docker PostGIS Authors (See AUTHORS)

# Permission is hereby granted, free of charge, to any person
# obtaining a copy of this software and associated documentation
# files (the "Software"), to deal in the Software without
# restriction, including without limitation the rights to use,
# copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the
# Software is furnished to do so, subject to the following
# conditions:

# The above copyright notice and this permission notice shall be
# included in all copies or substantial portions of the Software.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
# OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
# NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
# HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
# WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
# FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
# OTHER DEALINGS IN THE SOFTWARE.

FROM postgres:13-alpine

LABEL maintainer="PostGIS Project - https://postgis.net"

ENV POSTGIS_VERSION 3.1.2
ENV POSTGIS_SHA256 c49b6baa4afe4aed6cc7333399897aaf7540b40779a939a4d5a81d0725f6c9f8

#Temporary fix:
#   for PostGIS 2.* - building a special geos
#   reason:  PostGIS 2.5.5 is not working with GEOS 3.9.*
ENV POSTGIS2_GEOS_VERSION tags/3.8.2

RUN set -eux \
    \
    && apk add --no-cache --virtual .fetch-deps \
        ca-certificates \
        openssl \
        tar \
    \
    && wget -O postgis.tar.gz "https://github.com/postgis/postgis/archive/$POSTGIS_VERSION.tar.gz" \
    && echo "$POSTGIS_SHA256 *postgis.tar.gz" | sha256sum -c - \
    && mkdir -p /usr/src/postgis \
    && tar \
        --extract \
        --file postgis.tar.gz \
        --directory /usr/src/postgis \
        --strip-components 1 \
    && rm postgis.tar.gz \
    \
    && apk add --no-cache --virtual .build-deps \
        autoconf \
        automake \
        clang-dev \
        file \
        g++ \
        gcc \
        gdal-dev \
        gettext-dev \
        json-c-dev \
        libtool \
        libxml2-dev \
        llvm11-dev \
        make \
        pcre-dev \
        perl \
        proj-dev \
        protobuf-c-dev \
     \
# GEOS setup
     && if   [ $(printf %.1s "$POSTGIS_VERSION") == 3 ]; then \
            apk add --no-cache --virtual .build-deps-geos geos-dev cunit-dev ; \
        elif [ $(printf %.1s "$POSTGIS_VERSION") == 2 ]; then \
            apk add --no-cache --virtual .build-deps-geos cmake git ; \
            cd /usr/src ; \
            git clone https://github.com/libgeos/geos.git ; \
            cd geos ; \
            git checkout ${POSTGIS2_GEOS_VERSION} -b geos_build ; \
            mkdir cmake-build ; \
            cd cmake-build ; \
                cmake -DCMAKE_BUILD_TYPE=Release .. ; \
                make -j$(nproc) ; \
                make check ; \
                make install ; \
            cd / ; \
            rm -fr /usr/src/geos ; \
        else \
            echo ".... unknown PosGIS ...." ; \
        fi \
    \
# build PostGIS
    \
    && cd /usr/src/postgis \
    && gettextize \
    && ./autogen.sh \
    && ./configure \
        --with-pcredir="$(pcre-config --prefix)" \
    && make -j$(nproc) \
    && make install \
    \
# regress check
    && mkdir /tempdb \
    && chown -R postgres:postgres /tempdb \
    && su postgres -c 'pg_ctl -D /tempdb init' \
    && su postgres -c 'pg_ctl -D /tempdb start' \
    && cd regress \
    && make -j$(nproc) check RUNTESTFLAGS=--extension   PGUSER=postgres \
    #&& make -j$(nproc) check RUNTESTFLAGS=--dumprestore PGUSER=postgres \
    #&& make garden                                      PGUSER=postgres \
    && su postgres -c 'pg_ctl -D /tempdb --mode=immediate stop' \
    && rm -rf /tempdb \
    && rm -rf /tmp/pgis_reg \
# add .postgis-rundeps
    && apk add --no-cache --virtual .postgis-rundeps \
        gdal \
        json-c \
        libstdc++ \
        pcre \
        proj \
        protobuf-c \
     # Geos setup
     && if [ $(printf %.1s "$POSTGIS_VERSION") == 3 ]; then \
            apk add --no-cache --virtual .postgis-rundeps-geos geos ; \
        fi \
# clean
    && cd / \
    && rm -rf /usr/src/postgis \
    && apk del .fetch-deps .build-deps .build-deps-geos

COPY ./initdb-postgis.sh /docker-entrypoint-initdb.d/10_postgis.sh
COPY ./update-postgis.sh /usr/local/bin

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