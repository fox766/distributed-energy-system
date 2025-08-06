#!/bin/bash
docker rm -f fabric-mysql > /dev/null 2>&1
./network.sh down