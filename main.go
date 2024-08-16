package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	bolt "go.etcd.io/bbolt"
)

const contractAddress = "0x761d53b47334bee6612c0bd1467fb881435375b2"
const eventTopic = "0x3e54d0825ed78523037d00a81759237eb436ce774bd546993ee67a1b67b6e766"

type EventData struct {
	BlockTime  uint64
	ParentHash common.Hash
	L1InfoRoot common.Hash
}

func main() {
	client, err := ethclient.Dial("https://sepolia.infura.io/v3/07f0dfde071243bdbc4c3a53562536cf")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress(contractAddress)},
		Topics:    [][]common.Hash{{common.HexToHash(eventTopic)}},
	}

	logs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		log.Fatalf("Failed to retrieve logs: %v", err)
	}

	db, err := bolt.Open("events.db", 0600, nil)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	var index uint64
	for _, vLog := range logs {
		block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(vLog.BlockNumber)))
		if err != nil {
			log.Fatalf("Failed to retrieve block: %v", err)
		}

		data := EventData{
			BlockTime:  block.Time(),
			ParentHash: block.ParentHash(),
			L1InfoRoot: common.BytesToHash(vLog.Data),
		}

		err = db.Update(func(tx *bolt.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists([]byte("Events"))
			if err != nil {
				return err
			}

			dataBytes, err := json.Marshal(data)
			if err != nil {
				return err
			}

			return bucket.Put(itob(index), dataBytes)
		})

		if err != nil {
			log.Fatalf("Failed to write to database: %v", err)
		}

		index++
	}
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}
