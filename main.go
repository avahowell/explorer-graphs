package main

import (
	"fmt"
	"log"
	"os"

	"github.com/NebulousLabs/Sia/encoding"
	"github.com/NebulousLabs/Sia/modules"
	"github.com/NebulousLabs/Sia/types"
	"github.com/NebulousLabs/bolt"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

const explorerdb = "explorer.db"

var (
	bucketBlockFacts = []byte("BlockFacts")
	siaColor         = drawing.Color{R: 47, B: 230, G: 55, A: 255}
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

	var bins []float64
	var xaxis []float64
	bin := types.NewCurrency64(0)
	bincount := 0
	j := 0

	for i := len(blockfacts) - 1; i >= 0; i-- {
		fact := blockfacts[i]

		bin = bin.Add(fact.ActiveContractCost)
		if j == binSize {
			binint, err := bin.Div64(uint64(binSize)).Div(types.SiacoinPrecision).Uint64()
			if err != nil {
				log.Fatal(err)
			}
			bins = append(bins, float64(binint))
			xaxis = append(xaxis, float64(bincount))
			bin = types.NewCurrency64(0)
			j = 0
			bincount++
		} else {
			j++
		}
	}

	out, err := os.Create("data.png")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	graph := chart.Chart{
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
			Name:      "Active Contract Spending (SC)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				XValues: xaxis,
				YValues: bins,
				Style: chart.Style{
					Show:        true,
					StrokeWidth: 3.0,
					StrokeColor: siaColor,
				},
			},
		},
	}

	err = graph.Render(chart.PNG, out)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("chart.png generated")
}
