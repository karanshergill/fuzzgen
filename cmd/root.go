package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// supported modes for wordlist generation
var modes = map[string]bool{
	"general":     true,
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
	- general
  	- directories
  	- subdomains
	- files
	- parameters
	- extensions
	
Usage:
	fuzzgen -m general    # Generate a general wordlist
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, ok := modes[mode]; !ok {
			return errors.New("invalid mode: " + mode + " supported modes are: general, directories, subdomains, files, parameters, extensions)")
		}

		// print the urls for the given mode
		err := getSourceURLs("sources.yaml", mode)
		if err != nil {
			return err
		}

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

	// Debugging: Print the parsed sources map
	fmt.Printf("Parsed sources: %+v\n", sources)

	return sources, nil
}

func getSourceURLs(filename string, mode string) error {
	sources, err := getSourceMap(filename)
	if err != nil {
		return err
	}

	// get the urls for the given mode
	urls, ok := sources[mode]
	if !ok {
		return errors.New("no URLs found for mode: " + mode)
	}

	// print the urls
	fmt.Printf("URLs for mode '%s':\n", mode)
	for _, url := range urls {
		fmt.Println(url)
	}

	return nil

}

func init() {
	rootCmd.Flags().StringVarP(&mode, "mode", "m", "g", "Mode to generate wordlist. Supported modes: g (generic), d (directories), s (subdomains)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
