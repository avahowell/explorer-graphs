package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/NebulousLabs/Sia/encoding"
	"github.com/NebulousLabs/Sia/modules"
	"github.com/NebulousLabs/Sia/types"
	"github.com/NebulousLabs/bolt"
)

const explorerdb = "explorer.db"

var (
	bucketBlockFacts = []byte("BlockFacts")
)

type blockFacts struct {
	modules.BlockFacts

	Timestamp types.Timestamp
}

func getBlockFacts(db *bolt.DB) []blockFacts {
	var blockfacts []blockFacts
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketBlockFacts)
		c := b.Cursor()

		for k, blockfactbytes := c.First(); k != nil; k, blockfactbytes = c.Next() {
			var bf blockFacts
			err := encoding.Unmarshal(blockfactbytes, &bf)
			if err != nil {
				return err
			}
			blockfacts = append(blockfacts, bf)
		}
		return nil
	})
	return blockfacts
}

func main() {
	db, err := bolt.Open(explorerdb, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	blockfacts := getBlockFacts(db)
	binSize := 1008

	var bins []types.Currency
	bin := types.NewCurrency64(0)
	i := 0
	for _, fact := range blockfacts[:len(blockfacts)-binSize] {
		bin = bin.Add(fact.ActiveContractCost)
		if i == binSize {
			bins = append(bins, bin.Div64(uint64(binSize)))
			bin = types.NewCurrency64(0)
			i = 0
		} else {
			i++
		}
	}

	// convert bins to SC
	for i, bin := range bins {
		bins[i] = bin.Div(types.SiacoinPrecision)
	}

	// create our `data` javascript
	bytes, err := json.Marshal(bins)
	if err != nil {
		log.Fatal(err)
	}
	bytes = append([]byte("var data = "), bytes...)

	out, err := os.Create("data.json")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	_, err = out.Write(bytes)
	if err != nil {
		log.Fatal(err)
	}
	out.Close()

	err = os.Rename("data.json", "frontend/data.js")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("graphs successfully generated, open frontend/index.html")
}
