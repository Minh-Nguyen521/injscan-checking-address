package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

var ListNFT = []string{"Injective Quants", "The Ninjas"}

type SheetData struct {
	Range          string     `json:"range"`
	MajorDimension string     `json:"majorDimension"`
	Values         [][]string `json:"values"`
}

// func checkBalance(injAddress string) {

// }

// func checkTx(injAddress string, apiKey string) bool {
// 	return false
// }

func checkNft(injAddress string, apiKey string) bool {
	// Make API request to check INJ address
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.talis.art/tokens/"+injAddress+"?offset=0&limit=100", nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return false
	}

	req.Header.Add("x-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	// Read and parse response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return false
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		return false
	}

	if result["tokens"] == nil {
		return false
	}

	for _, token := range result["tokens"].([]interface{}) {
		tokenMap := token.(map[string]interface{})
		family := tokenMap["family"].(map[string]interface{})
		for _, nft := range ListNFT {
			if family["name"] == nft {
				return true
			}
		}
	}
	return false
}

func main() {

	fmt.Println("Gmail and INJ Address Scanner - Reading")

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

	apiKey := os.Getenv("TALIS_API_KEY")
	if apiKey == "" {
		fmt.Println("TALIS_API_KEY not set in .env")
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
	totalRecords := len(sheetData.Values) - 1

	fmt.Println("\nScanning registered addresses:")
	fmt.Println("------------------------------------")

	// Create a file to store results
	resultsFile := "results.json"
	results := struct {
		Addresses []string `json:"addresses"`
	}{
		Addresses: make([]string, 0),
	}

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

	// Extract just the addresses from data
	for _, row := range sheetData.Values[1:] {
		if len(row) >= 2 {
			results.Addresses = append(results.Addresses, row[1])
		}
	}

	var addresses []string

	for _, row := range sheetData.Values[1:] {
		if len(row) < 2 {
			continue
		}

		// email := row[0]
		injAddress := row[1]

		// check balance of injAddress

		// check tx of injAddress
		// tx := checkTx(injAddress, apiKey)
		// if tx {
		// 	fmt.Println("TX found - ", injAddress)
		// 	addresses = append(addresses, injAddress)
		// 	continue
		// }

		// check nft of injAddress
		nft := checkNft(injAddress, apiKey)
		if nft {
			fmt.Println("NFT found - ", injAddress)
			addresses = append(addresses, injAddress)
		}

		// fmt.Printf("\nAddress: %s\n", injAddress)
		// fmt.Printf("Response:\n%s\n", pretty.Pretty(body))
	}

	// Write results to file
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(addresses); err != nil {
		fmt.Printf("Error writing results: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nSummary:")
	fmt.Printf("Total records scanned: %d\n", totalRecords)
}
