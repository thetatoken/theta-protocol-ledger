#!/bin/bash

# Build a docker image for a Theta node.
# Usage: 
#    integration/docker/node/build.sh
#
# After the image is built, you can create a container by:
#    docker stop theta_node
#    docker rm theta_node
#    docker run -e THETA_CONFIG_PATH=/theta/integration/privatenet/node --name theta_node -it theta
set -e

SCRIPTPATH=$(dirname "$0")

echo $SCRIPTPATH

if [ "$1" =  "force" ] || [[ "$(docker images -q theta 2> /dev/null)" == "" ]]; then
    docker build -t theta -f $SCRIPTPATH/Dockerfile .
fi


