#!/bin/bash
docker rm -f fabric_mysql > /dev/null 2>&1
./network.sh down