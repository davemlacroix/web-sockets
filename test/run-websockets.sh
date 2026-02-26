#!/bin/bash

docker build --tag websockets ../src

docker run  \
    -p 127.0.0.1:9001:9001 \
    --network=host \
    websockets