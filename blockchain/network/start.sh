#!/bin/bash
./network.sh down
 ./network.sh up createChannel -c mychannel -ca
./network.sh deployCC -ccn basic -ccp ../chaincode-go -ccl go