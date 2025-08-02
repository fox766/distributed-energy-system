#!/bin/bash
./network.sh down
./network.sh up createChannel
./network.sh deployCC -ccn energySys -ccp /home/fox766/go/src/github.com/fox766/fabric-samples/DistrEnergySys/distributed-energy-system/blockchain/chaincode -ccl go