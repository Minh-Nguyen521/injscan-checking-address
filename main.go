package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type SheetData struct {
	Range          string     `json:"range"`
	MajorDimension string     `json:"majorDimension"`
	Values         [][]string `json:"values"`
}

func main() {
	fmt.Println("Gmail and INJ Address Scanner - Reading from registered.json")

	// Read the JSON file
	jsonFile, err := ioutil.ReadFile("registered.json")
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

	for i, row := range sheetData.Values[1:] {
		if len(row) < 2 {
			continue
		}

		email := row[0]
		injAddress := row[1]

		// Print validation results
		fmt.Printf("Record #%d:\n", i+1)
		fmt.Printf("  Email: %s \n", email)
		fmt.Printf("  INJ:   %s \n", injAddress)
		fmt.Println()
	}

	fmt.Println("\nSummary:")
	fmt.Printf("Total records scanned: %d\n", totalRecords)
}
