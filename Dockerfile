FROM postgres:13

RUN apt-get update \
    && apt-get install postgis postgresql-13-postgis-3 -y

COPY init-db.sh /docker-entrypoint-initdb.d/

