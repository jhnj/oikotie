#!/bin/sh

set -euo pipefail

docker container kill $(docker container ls -qf ancestor=oikotie_pg:1)
docker run --publish 5432:5432 -e POSTGRES_PASSWORD=password --detach oikotie_pg:1