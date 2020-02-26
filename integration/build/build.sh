#!/bin/bash

# Usage: 
#    integration/build/build.sh
#    integration/build/build.sh force # Always recreate docker image and container.
set -e

SCRIPTPATH=$(dirname "$0")

echo $SCRIPTPATH

if [ "$1" =  "force" ] || [[ "$(docker images -q theta_builder_image 2> /dev/null)" == "" ]]; then
    docker build -t theta_builder_image $SCRIPTPATH
fi

set +e
docker stop theta_builder
docker rm theta_builder
set -e

docker run --name theta_builder -it -v "$GOPATH:/go" theta_builder_image
