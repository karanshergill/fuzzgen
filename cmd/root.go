package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var mode string

// supported modes for wordlist generation
var modes = map[string]string{
	"g": "general",
	"d": "directories",
	"f": "files",
	"p": "parameters",
	"e": "extensions",
	"s": "subdomains",
}
var rootCmd = &cobra.Command{
	Use:   "fuzzgen",
	Short: "Generate wordlists for web fuzzing",
	Long: `A fast and simple tool to generate wordlists for web fuzzing.
	
Wordlist Generation Modes:
	- g: general
  	- d: directorues
  	- s: subdomains
	- f: files
	- p: parameters
	- e: extensions
	
Usage:
	fuzzgen -m g    # Generate a general wordlist
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, ok := modes[mode]; !ok {
			return errors.New("Invalid mode:" + mode)
		}
		switch mode {
		case "g":
			fmt.Println("Generating general wordlist")
			generalWordlist()
		case "d":
			fmt.Println("Generating directories wordlist")
			directoriesWordlist()
		case "f":
			fmt.Println("Generating files wordlist")
		case "p":
			fmt.Println("Generating parameters wordlist")
		case "e":
			fmt.Println("Generating extensions wordlist")
		case "s":
			fmt.Println("Generating subdomains wordlist")
		}
		return nil
	},
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

func generalWordlist() {
	fmt.Println("started...")
}

func directoriesWordlist() {
	fmt.Println("started...")
}
