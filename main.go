package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const mitoContractAddress = "inj1vcqkkvqs7prqu70dpddfj7kqeqfdz5gg662qs3"

var ListNFT = []string{"quant", "ninja"}
var ListContractAddress = []string{"inj1vtd54v4jm50etkjepgtnd7lykr79yvvah8gdgw", "inj19ly43dgrr2vce8h02a8nw0qujwhrzm9yv8d75c"}

type SheetData struct {
	Range          string     `json:"range"`
	MajorDimension string     `json:"majorDimension"`
	Values         [][]string `json:"values"`
}

func checkNft(injAddress string, rpcUrl string) []string {
	queryMsg := interface{}(map[string]interface{}{
		"tokens": map[string]interface{}{
			"owner": injAddress,
		},
	})

	queryjson, err := json.Marshal(queryMsg)
	if err != nil {
		fmt.Printf("Error marshalling query message: %v\n", err)
		return []string{}
	}

	queryBzStr := base64.StdEncoding.EncodeToString(queryjson)

	var nft []string
	for i, contractAddress := range ListContractAddress {
		url := fmt.Sprintf("%s/cosmwasm/wasm/v1/contract/%s/smart/%s", rpcUrl, contractAddress, queryBzStr)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error making HTTP request: %v\n", err)
			return []string{}
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response body: %v\n", err)
			return []string{}
		}

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			return []string{}
		}

		var data map[string]interface{}
		if response["data"] != nil {
			data = response["data"].(map[string]interface{})
		}

		if ids, ok := data["ids"].([]interface{}); ok && len(ids) > 0 {
			nft = append(nft, ListNFT[i])
		}
	}
	return nft
}

func checkBalance(injAddress string, rpcUrl string) int {

	url := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", rpcUrl, injAddress)

	resp, err := http.Get(url)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0
	}

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0
	}

	balances, ok := response["balances"].([]interface{})
	if !ok {
		return 0
	}

	for _, balance := range balances {
		balanceMap, ok := balance.(map[string]interface{})
		if !ok {
			continue
		}
		if balanceMap["denom"] == "inj" {
			amount, ok := balanceMap["amount"].(string)
			if !ok {
				return 0
			}
			amountInt, err := strconv.Atoi(amount)
			if err != nil {
				return 0
			}
			return amountInt
		}
	}
	return 0
}

func checkDex(injAddress string, indexerUrl string) (bool, bool) {
	url := fmt.Sprintf("%s/api/explorer/v1/accountTxs/%s", indexerUrl, injAddress)

	resp, err := http.Get(url)
	if err != nil {
		return false, false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, false
	}

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return false, false
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		return false, false
	}

	var flagHelix bool = false
	var flagMito bool = false

	for _, item := range data {
		// check mito contract
		if !flagMito {
			messages := item.(map[string]interface{})["messages"].([]interface{})
			value := messages[0].(map[string]interface{})["value"].(map[string]interface{})
			contractAddress := ""
			if contractAddr, ok := value["contract_address"]; ok {
				contractAddress = contractAddr.(string)
			}

			if contractAddress == mitoContractAddress {
				flagMito = true
			}
		}

		// check helix's transactions
		logs, ok := item.(map[string]interface{})["logs"].([]interface{})
		if !ok || len(logs) == 0 {
			continue
		}
		if !flagHelix {
			for _, log := range logs {
				logMap, ok := log.(map[string]interface{})
				if !ok {
					continue
				}
				eventMap, ok := logMap["events"].([]interface{})
				if !ok || len(eventMap) == 0 {
					continue
				}
				for _, event := range eventMap {
					eventMap, ok := event.(map[string]interface{})
					if !ok {
						continue
					}
					eventType, ok := eventMap["type"].(string)
					if !ok {
						continue
					}
					if strings.Contains(eventType, "injective.exchange.v1beta1.") {
						flagHelix = true
						break
					}
				}
				if flagHelix {
					break
				}
			}
		}
		if flagHelix && flagMito {
			return true, true
		}
	}

	return flagHelix, flagMito
}

type SellOrdersResponse struct {
	Data struct {
		Orders []struct {
			Owner           string `json:"owner"`
			ContractAddress string `json:"contract_address"`
		} `json:"orders"`
	} `json:"data"`
}

func main() {

	fmt.Println("Starting...")
	// Get filename from environment variable
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
		os.Exit(1)
	}

	filename := os.Getenv("REGISTERED_FILE")
	if filename == "" {
		fmt.Println("REGISTERED_FILE not set in .env")
		os.Exit(1)
	}

	rpcUrl := os.Getenv("RPC_URL")
	if rpcUrl == "" {
		fmt.Println("RPC_URL not set in .env")
		os.Exit(1)
	}

	indexerUrl := os.Getenv("INDEXER_URL")
	if indexerUrl == "" {
		fmt.Println("INDEXER_URL not set in .env")
		os.Exit(1)
	}

	// Read the JSON file
	jsonFile, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse JSON data
	var sheetData SheetData
	err = json.Unmarshal(jsonFile, &sheetData)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Skip the header row
	if len(sheetData.Values) < 2 {
		fmt.Println("No data found in the sheet")
		os.Exit(1)
	}

	// Create a file to store results
	resultsFile := "results.json"

	// Remove existing file if it exists
	if _, err := os.Stat(resultsFile); err == nil {
		if err := os.Remove(resultsFile); err != nil {
			fmt.Printf("Error removing existing results file: %v\n", err)
			os.Exit(1)
		}
	}

	// Create new results file
	f, err := os.Create(resultsFile)
	if err != nil {
		fmt.Printf("Error creating results file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	fmt.Println("Scanning sell orders...")
	// Make HTTP request to get sell orders

	url := fmt.Sprintf("%s/cosmwasm/wasm/v1/contract/inj1l9nh9wv24fktjvclc4zgrgyzees7rwdtx45f54/smart/eyJhbGxfc2VsbF9vcmRlcnMiOnt9fQ==", rpcUrl)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		os.Exit(1)
	}

	var response SellOrdersResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	sellOrders := make(map[string][]string)

	for _, order := range response.Data.Orders {
		sellOrders[order.Owner] = append(sellOrders[order.Owner], order.ContractAddress)
	}

	type AddressNFTs struct {
		Addresses  string   `json:"addresses"`
		InjBalance int      `json:"inj_balance"`
		Nfts       []string `json:"nfts"`
		Helix      bool     `json:"helix"`
		Mito       bool     `json:"mito"`
	}

	var data []AddressNFTs

	fmt.Println("Scanning nft and sell orders...")
	for i, row := range sheetData.Values[1:] {
		if len(row) < 2 {
			continue
		}

		injAddress := row[1]

		listNft := make(map[string]bool)
		// Check sell orders
		if orders, hasOrders := sellOrders[injAddress]; hasOrders {
			for i, contractAddr := range ListContractAddress {
				for _, sellOrder := range orders {
					if sellOrder == contractAddr {
						listNft[ListNFT[i]] = true
						break
					}
				}
			}
		}

		// Check NFT ownership
		if nfts := checkNft(injAddress, rpcUrl); len(nfts) > 0 {
			for _, nft := range nfts {
				listNft[nft] = true
			}
		}

		result := AddressNFTs{
			Addresses:  injAddress,
			InjBalance: 0,
			Nfts:       []string{},
			Helix:      false,
			Mito:       false,
		}

		var nftList []string
		if len(listNft) > 0 {
			for nft := range listNft {
				nftList = append(nftList, nft)
			}

			result.Nfts = nftList
			result.Addresses = injAddress
		}

		// check balance
		balance := checkBalance(injAddress, rpcUrl)
		if balance > 0 {
			result.InjBalance = balance

			// check dex
			flagHelix, flagMito := checkDex(injAddress, indexerUrl)
			result.Helix = flagHelix
			result.Mito = flagMito
			result.Addresses = injAddress
		}

		if len(result.Nfts) > 0 || result.Helix || result.Mito {
			data = append(data, result)
		}

		if i%5 == 0 {
			time.Sleep(time.Second)
		}
	}

	// Write results to file
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(data); err != nil {
		fmt.Printf("Error writing results: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done")
}
