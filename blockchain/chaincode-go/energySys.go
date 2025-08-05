/*
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"log"
	
	"github.com/fox766/fabric-samples/DistrEnergySys/distributed-energy-system/blockchain/chaincode-go/chaincode"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

func main() {
	assetChaincode, err := contractapi.NewChaincode(&chaincode.SmartContract{})
	if err != nil {
		log.Panicf("Error creating energySystem chaincode: %v", err)
	}

	if err := assetChaincode.Start(); err != nil {
		log.Panicf("Error starting  energySystem chaincode: %v", err)
	}
}
