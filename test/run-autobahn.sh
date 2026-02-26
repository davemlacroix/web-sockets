#!/bin/bash

docker run -it --rm \
    -v "${PWD}/config:/config" \
    -v "${PWD}/reports:/reports" \
    -p 9001:9001 \
    --name fuzzingclient \
    crossbario/autobahn-testsuite:25.10.1