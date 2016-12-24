package main

import (
	"flag"
	"log"
	"os"

	"github.com/NebulousLabs/Sia/encoding"
	"github.com/NebulousLabs/Sia/modules"
	"github.com/NebulousLabs/Sia/types"
	"github.com/NebulousLabs/bolt"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

var (
	bucketBlockFacts = []byte("BlockFacts")
	siaColor         = drawing.Color{R: 47, G: 230, B: 55, A: 255}

	explorerdb = flag.String("db", "explorer.db", "path to the Sia explorer bolt database")
	outpath    = flag.String("out", "out.png", "save path for the generated graph")
)

type blockFacts struct {
	modules.BlockFacts

	Timestamp types.Timestamp
}

// getBlockFacts walks through the explorer database and returns a slice of
// blockFacts, where blockfacts[0] is the blockFacts for the first block on the
// blockchain, and blockfacts[len(blockfacts)-1] is the blockFacts for the last
// block on the blockchain.
func getBlockFacts(db *bolt.DB) ([]blockFacts, error) {
	var blockfacts []blockFacts
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketBlockFacts)
		c := b.Cursor()

		for k, blockfactbytes := c.Last(); k != nil; k, blockfactbytes = c.Prev() {
			var bf blockFacts
			err := encoding.Unmarshal(blockfactbytes, &bf)
			if err != nil {
				return err
			}
			blockfacts = append(blockfacts, bf)
		}
		return nil
	})
	return blockfacts, err
}

// activeContractGraph creates a go-chart graph of active contract spending
// given a slice of block facts.
func activeContractGraph(bf []blockFacts) (*chart.Chart, error) {
	// use a bin size of 1008, or 1 block-week.
	binSize := 1008

	var yaxis []float64
	var xaxis []float64
	bin := types.NewCurrency64(0)

	j := 0
	for i := 0; i < len(bf); i++ {
		fact := bf[i]

		bin = bin.Add(fact.ActiveContractCost)
		if j == binSize {
			binint, err := bin.Div64(uint64(binSize)).Div(types.SiacoinPrecision).Uint64()
			if err != nil {
				return nil, err
			}

			yaxis = append(yaxis, float64(binint))
			xaxis = append(xaxis, float64(len(yaxis)))
			bin = types.NewCurrency64(0)
			j = 0
		} else {
			j++
		}
	}

	return &chart.Chart{
		Title: "Active Contract Spending Over Time",
		TitleStyle: chart.Style{
			Show: true,
		},
		Width:  800,
		Height: 500,
		Background: chart.Style{
			Padding: chart.Box{
				Top:    100,
				Left:   5,
				Right:  5,
				Bottom: 5,
			},
		},
		XAxis: chart.XAxis{
			Name:      "Block Height (thousands)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      "Active Contract Spending (Million SC)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				XValues: xaxis,
				YValues: yaxis,
				Style: chart.Style{
					Show:        true,
					StrokeWidth: 3.0,
					StrokeColor: siaColor,
				},
			},
		},
	}, nil
}

func main() {
	flag.Parse()

	db, err := bolt.Open(*explorerdb, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	blockfacts, err := getBlockFacts(db)
	if err != nil {
		log.Fatal(err)
	}

	graph, err := activeContractGraph(blockfacts)
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.Create(*outpath)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	err = graph.Render(chart.PNG, out)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("rendered graph from %v to %v\n", *explorerdb, *outpath)
}
