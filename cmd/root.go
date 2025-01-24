package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// supported modes for wordlist generation
var modes = map[string]bool{
	"generic":     true,
	"directories": true,
	"files":       true,
	"parameters":  true,
	"extensions":  true,
	"subdomains":  true,
}

var mode string

var rootCmd = &cobra.Command{
	Use:   "fuzzgen",
	Short: "Generate wordlists for web fuzzing",
	Long: `A fast and simple tool to generate wordlists for web fuzzing.
	
Wordlist Generation Modes:
	- generic
  	- directories
  	- subdomains
	- files
	- parameters
	- extensions
	
Usage:
	fuzzgen -m generic    # Generate a general wordlist
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, ok := modes[mode]; !ok {
			return errors.New("invalid mode: " + mode + " supported modes are: generic, directories, subdomains, files, parameters, extensions)")
		}

		// print the urls for the given mode
		sourceUrls, err := getSourceUrls("sources.yaml", mode)
		if err != nil {
			return err
		}

		// Fetch and process data from all URLs
		var allLines []string
		for _, url := range sourceUrls {
			lines, err := fetchDataFromSourceUrls(url)
			if err != nil {
				fmt.Printf("Error fetching data from %s: %v\n", url, err)
				continue
			}
			allLines = append(allLines, lines...)
		}

		// Deduplicate the lines using hashing
		uniqueLines := deduplicateLines(allLines)

		// Save the generated wordlist to a file named <mode>.txt
		filename := mode + ".txt"
		err = ioutil.WriteFile(filename, []byte(strings.Join(uniqueLines, "\n")), 0644)
		if err != nil {
			return fmt.Errorf("error writing wordlist to file: %v", err)
		}

		// Print success message
		fmt.Printf("Generated wordlist saved to %s\n", filename)

		return nil
	},
}

func getSourceMap(filename string) (map[string][]string, error) {
	fmt.Println("fetching sources...")

	// read the source file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.New("error reading source file: " + err.Error())
	}

	// parse the sources.yaml file into a map to get the urls
	var sources map[string][]string
	err = yaml.Unmarshal(data, &sources)
	if err != nil {
		return nil, errors.New("error unmarshalling source file: " + err.Error())
	}

	// print the parsed sources map
	fmt.Printf("Parsed sources: %+v\n", sources)

	return sources, nil
}

func getSourceUrls(filename string, mode string) ([]string, error) {
	sources, err := getSourceMap(filename)
	if err != nil {
		return nil, err
	}

	// get the urls for the given mode
	sourceUrls, ok := sources[mode]
	if !ok {
		return nil, errors.New("no URLs found for mode: " + mode)
	}

	// print the urls
	fmt.Printf("URLs for mode '%s':\n", mode)
	for _, url := range sourceUrls {
		fmt.Println(url)
	}

	return sourceUrls, nil

}

func fetchDataFromSourceUrls(sourceUrl string) ([]string, error) {

	var lines []string

	resp, err := http.Get(sourceUrl)
	if err != nil {
		return nil, errors.New("error fetching data from source: " + err.Error())
	}
	defer resp.Body.Close()

	// check response status code
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("error fetching data from source: " + sourceUrl + ": " + resp.Status)
	}

	// read the response line by line
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	// check for scanning errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading response from URL %s: %v", sourceUrl, err)
	}

	return lines, nil
}

func deduplicateLines(lines []string) []string {
	uniqueLines := make([]string, 0)
	seen := make(map[string]bool)

	for _, line := range lines {
		if !seen[line] {
			seen[line] = true
			uniqueLines = append(uniqueLines, line)
		}
	}

	return uniqueLines
}

func init() {
	rootCmd.Flags().StringVarP(&mode, "mode", "m", "generic", "Supported modes: g (generic), d (directories), s (subdomains)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
