package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

var ListNFT = []string{"Injective Quants", "The Ninjas"}
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

	sellOrders := make(map[string]string)

	for _, order := range response.Data.Orders {
		sellOrders[order.Owner] = order.ContractAddress
	}

	var data struct {
		Addresses []string   `json:"addresses"`
		NFTs      [][]string `json:"nfts"`
	}

	fmt.Println("Scanning nft and sell orders...")
	for i, row := range sheetData.Values[1:10] {
		if len(row) < 2 {
			continue
		}

		injAddress := row[1]

		// check sell orders of injAddress
		if _, ok := sellOrders[injAddress]; ok {
			var flag bool = false
			for i, contractAddress := range ListContractAddress {
				if sellOrders[injAddress] == contractAddress {
					data.Addresses = append(data.Addresses, injAddress)
					data.NFTs = append(data.NFTs, []string{injAddress, ListNFT[i]})
					flag = true
					break
				}
			}
			if flag {
				continue
			}
		}

		// check nft of injAddress
		nft := checkNft(injAddress, rpcUrl)
		if len(nft) > 0 {
			data.Addresses = append(data.Addresses, injAddress)
			data.NFTs = append(data.NFTs, nft)
		}

		if i%5 == 0 {
			time.Sleep(1 * time.Second)
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
