package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// SourceURLs structure to parse YAML
type SourceURLs struct {
	Subdomains  []string `yaml:"subdomains"`
	Directories []string `yaml:"directories"`
}

// Function to parse the sources YAML file
func parseSourcesFile(filepath string) (SourceURLs, error) {
	var urls SourceURLs

	// Read the sources file
	fileContent, err := os.ReadFile(filepath)
	if err != nil {
		return urls, err
	}

	// Unmarshall the coontent
	err = yaml.Unmarshal(fileContent, &urls)
	if err != nil {
		return urls, err
	}

	return urls, nil
}

// Function to get the status code of urls
func getSourceData(url string) (int, string, error) {
	response, err := http.Get(url)
	if err != nil {
		return 0, "", err
	}
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)

	if err != nil {
		return response.StatusCode, "", err
	}

	body := string(bodyBytes)

	return response.StatusCode, body, nil
}

func processSourceURLs(category string, urls SourceURLs) {
	switch category {
	case "subdomains":
		fmt.Println("Checking Subdomains:")
		for _, url := range urls.Subdomains {
			statusCode, body, err := getSourceData(url)
			if err != nil {
				fmt.Printf("Error fetching URL %s: %v\n", url, err)
			} else {
				body = strings.ToLower(body)
				fmt.Printf("URL: %s, Status Code: %d, Response Body: %s\n", url, statusCode, body)
			}
		}
	case "directories":
		fmt.Println("\nChecking Directories:")
		for _, url := range urls.Directories {
			statusCode, body, err := getSourceData(url)
			if err != nil {
				fmt.Printf("Error fetching URL %s: %v\n", url, err)
			} else {
				fmt.Printf("URL: %s, Status Code: %d, Response Body: %s\n", url, statusCode, body)
			}
		}
	case "all":
		processSourceURLs("subdomains", urls)
		processSourceURLs("directories", urls)
	default:
		fmt.Println("Invalid category. Please choose 'subdomains', 'directories', or 'all'.")
	}
}

func main() {
	// Static path to the YAML file
	filePath := "sources.yaml"

	var category string

	// Create a new root command
	var rootCmd = &cobra.Command{
		Use:   "fuzzgen",
		Short: "Fuzzgen processes URLs from a YAML file and fetches their HTTP status",
		Run: func(cmd *cobra.Command, args []string) {
			if category == "" {
				fmt.Println("Error: The category flag (-c) is required.")
				cmd.Usage()
				os.Exit(1)
			}
			// Parse the YAML file
			urls, err := parseSourcesFile(filePath)
			if err != nil {
				fmt.Printf("Error parsing YAML file: %v\n", err)
				os.Exit(1)
			}

			// Process the chosen category
			processSourceURLs(category, urls)
		},
	}

	// Add a flag for the category
	rootCmd.Flags().StringVarP(&category, "category", "c", "", "Category to process: subdomains, directories, or all")

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
