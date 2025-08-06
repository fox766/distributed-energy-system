#!/bin/bash
./stop.sh
docker run --name fabric_mysql -p 3337:3306 -e MYSQL_ROOT_PASSWORD=fabric -d mysql:8
 ./network.sh up createChannel -c mychannel -ca
./network.sh deployCC -ccn basic -ccp ../chaincode-go -ccl go