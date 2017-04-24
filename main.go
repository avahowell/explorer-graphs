package main

import (
	"flag"
	"log"
	"os"
	"sort"

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

	defaultGraphs = []graphParams{
		{func(bf blockFacts) types.Currency { return bf.ActiveContractSize.Div64(1e9) }, "Active Contract Size", "Contract Size (GB)"},
		{func(bf blockFacts) types.Currency { return bf.ActiveContractCost.Div(types.SiacoinPrecision) }, "Active Contract Cost", "Contract Cost (SC)"},
		{func(bf blockFacts) types.Currency { return bf.TotalContractCost.Div(types.SiacoinPrecision) }, "Total Contract Cost", "Total Contract Cost (SC)"},
		{func(bf blockFacts) types.Currency { return bf.TotalContractSize.Div64(1e9) }, "Total Contract Size", "Total Contract Size (GB)"},
	}
)

type (
	blockFacts struct {
		modules.BlockFacts

		Timestamp types.Timestamp
	}

	factSlice   []blockFacts
	factGetter  func(bf blockFacts) types.Currency
	graphParams struct {
		fg     factGetter
		title  string
		ylabel string
	}
)

// getBlockFacts walks through the explorer database and returns a slice of
// blockFacts, where blockfacts[0] is the blockFacts for the first block on the
// blockchain, and blockfacts[len(blockfacts)-1] is the blockFacts for the last
// block on the blockchain.
func getBlockFacts(db *bolt.DB) (factSlice, error) {
	var blockfacts factSlice
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

	sort.Sort(blockfacts)

	return blockfacts, err
}

func (bf factSlice) Len() int           { return len(bf) }
func (bf factSlice) Less(i, j int) bool { return bf[i].Height < bf[j].Height }
func (bf factSlice) Swap(i, j int)      { bf[i], bf[j] = bf[j], bf[i] }

// Graph graphs block fact data received by the provded factGetter and returns
// a chart labelled acording to the `title` and `ylabel`.
func (bf factSlice) Graph(params graphParams) (*chart.Chart, error) {
	var yaxis []float64
	var xaxis []float64

	for i, fact := range bf {
		val, err := params.fg(fact).Uint64()
		if err != nil {
			return nil, err
		}

		yaxis = append(yaxis, float64(val))
		xaxis = append(xaxis, float64(i))
	}

	return &chart.Chart{
		Title: params.title,
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
			Name:      "Block Height",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
			Ticks: []chart.Tick{
				{0.0, "0"},
				{50000.0, "50k"},
				{100000.0, "100k"},
				{xaxis[len(xaxis)-1], ""},
			},
		},
		YAxis: chart.YAxis{
			Name:      params.ylabel,
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

	for _, param := range defaultGraphs {
		graph, err := blockfacts.Graph(param)
		if err != nil {
			log.Fatal(err)
		}

		savepath := param.title + ".png"

		out, err := os.Create(savepath)
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()

		err = graph.Render(chart.PNG, out)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("rendered graph from %v to %v\n", *explorerdb, savepath)
	}
}
