package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	allWatched  []*Watched
	port        string
	prefix      string
	loadSeconds float64
	totalLoaded int64
	eth         *ethclient.Client
)

type Watched struct {
	Name    string
	Address string
	Balance string
}

// ConnectionToGeth Connect to Geth server
func ConnectionToGeth(url string) error {
	var err error
	eth, err = ethclient.Dial(url)
	if err != nil {
		fmt.Printf("Error connecting to Geth on url: %v\n", url)
	}
	return err
}

// WeiToEther WEI to ETH conversion
func weiToEther(wei *big.Int) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(params.Ether))
}

// GetEthBalance Fetch ETH balance for an address from Geth server
func GetEthBalance(address string) *big.Float {
	balance, err := eth.BalanceAt(context.TODO(), common.HexToAddress(address), nil)
	if err != nil {
		fmt.Printf("Error fetching ETH Balance for address: %v\n", address)
	}
	return weiToEther(balance)
}

// CurrentBlock Fetch current block from Geth server
func CurrentBlock() uint64 {
	block, err := eth.BlockByNumber(context.TODO(), nil)
	if err != nil {
		fmt.Printf("Error fetching current block height: %v\n", err)
		return 0
	}
	return block.NumberU64()
}

// OpenAddresses Open the addresses.txt file and load all addresses into memory
func OpenAddresses(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		object := strings.Split(scanner.Text(), ":")
		if common.IsHexAddress(object[1]) {
			w := &Watched{
				Name:    object[0],
				Address: object[1],
			}
			allWatched = append(allWatched, w)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return err
}

// MetricsHttp HTTP response handler for /metrics endpoint
func MetricsHttp(w http.ResponseWriter, r *http.Request) {
	var allOut []string
	total := big.NewFloat(0)
	for _, v := range allWatched {
		if v.Balance == "" {
			v.Balance = "0"
		}
		bal := big.NewFloat(0)
		bal.SetString(v.Balance)
		total.Add(total, bal)
		allOut = append(allOut, fmt.Sprintf("%veth_balance{name=\"%v\",address=\"%v\"} %v", prefix, v.Name, v.Address, v.Balance))
	}
	allOut = append(allOut, fmt.Sprintf("%veth_balance_total %0.18f", prefix, total))
	allOut = append(allOut, fmt.Sprintf("%veth_load_seconds %0.2f", prefix, loadSeconds))
	allOut = append(allOut, fmt.Sprintf("%veth_loaded_addresses %v", prefix, totalLoaded))
	allOut = append(allOut, fmt.Sprintf("%veth_total_addresses %v", prefix, len(allWatched)))
	fmt.Fprintln(w, strings.Join(allOut, "\n"))
}

func main() {
	prefix = os.Getenv("PREFIX")
	gethUrl := os.Getenv("GETH")
	port = os.Getenv("PORT")

	err := OpenAddresses("addresses.txt")
	if err != nil {
		panic(err)
	}

	err = ConnectionToGeth(gethUrl)
	if err != nil {
		panic(err)
	}

	// check address balances
	go func() {
		for {
			totalLoaded = 0
			t1 := time.Now()
			fmt.Printf("Checking %v wallets...\n", len(allWatched))
			for _, v := range allWatched {
				v.Balance = GetEthBalance(v.Address).String()
				totalLoaded++
			}
			t2 := time.Now()
			loadSeconds = t2.Sub(t1).Seconds()
			fmt.Printf("Finished checking %v wallets in %0.0f seconds, sleeping for %v seconds.\n", len(allWatched), loadSeconds, 15)
			time.Sleep(15 * time.Second)
		}
	}()

	block := CurrentBlock()

	fmt.Printf("ETHbalance has started on port %v using Geth server: %v at block #%v\n", port, gethUrl, block)
	http.HandleFunc("/metrics", MetricsHttp)
	panic(http.ListenAndServe("0.0.0.0:"+port, nil))
}
