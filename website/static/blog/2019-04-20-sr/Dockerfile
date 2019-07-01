FROM jupyter/base-notebook:latest

USER root

RUN apt-get update && \
	apt-get install -y binutils libproj-dev gdal-bin libspatialindex-dev

USER jovyan

RUN pip install geopandas scipy matplotlib numpy pandas shapely fiona \
  six pyproj geopy psycopg2-binary rtree descartes pysal xlrd
