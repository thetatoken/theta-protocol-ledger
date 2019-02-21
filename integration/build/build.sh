#!/bin/bash

set -e

SCRIPTPATH=$(dirname "$0")

echo $SCRIPTPATH

if [[ "$(docker images -q theta_builder 2> /dev/null)" == "" ]]; then
    docker build -t theta_builder $SCRIPTPATH
fi

docker run -it -v "$GOPATH:/go" theta_builder

