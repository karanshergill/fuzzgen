package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type SourceURLs struct {
	Subdomains []string `yaml:"subdomains"`
}

// function to parse the sources.yaml file
func parseSourcesYAML(filepath string) (map[string]bool, error) {
	var urls SourceURLs
	subdomainSourceURLs := make(map[string]bool)

	fileContent, err := os.ReadFile(filepath)
	if err != nil {
		return subdomainSourceURLs, err
	}
	err = yaml.Unmarshal(fileContent, &urls)
	if err != nil {
		return subdomainSourceURLs, err
	}

	for _, url := range urls.Subdomains {
		if _, exists := subdomainSourceURLs[url]; !exists {
			subdomainSourceURLs[url] = true
		}
	}

	return subdomainSourceURLs, err
}

func validateSourceURLs(urls map[string]bool) {
	client := &http.Client{Timeout: 30 * time.Second}
	invalidSourceURLs := []string{}

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"

	for url := range urls {
		req, err := http.NewRequest("HEAD", url, nil)
		if err != nil {
			fmt.Printf("Error creating request for URL %s: %v\n", url, err)
			invalidSourceURLs = append(invalidSourceURLs, url)
			continue
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error fetching URL %s: %v\n", url, err)
			invalidSourceURLs = append(invalidSourceURLs, url)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Invalid status code %d for URL %s\n", resp.StatusCode, url)
			invalidSourceURLs = append(invalidSourceURLs, url)
		}
		resp.Body.Close()
	}

	for _, url := range invalidSourceURLs {
		delete(urls, url)
	}
}

func processResponseBody(body io.Reader) <-chan string {
	output := make(chan string)
	go func() {
		scanner := bufio.NewScanner(body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			line = strings.TrimFunc(line, func(r rune) bool {
				return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
			})

			if line != "" {
				output <- strings.ToLower(line)
			}
		}
		close(output)
	}()
	return output
}

func storeToBadgerDB(db *badger.DB, lines <-chan string, url string) error {
	const batchSize = 1000
	var count int
	txn := db.NewTransaction(true)
	defer txn.Discard()

	for line := range lines {
		if _, err := txn.Get([]byte(line)); err == badger.ErrKeyNotFound {
			if err := txn.Set([]byte(line), []byte(url)); err != nil {
				return err
			}
			count++
		}

		if count >= batchSize {
			if err := txn.Commit(); err != nil {
				return fmt.Errorf("error committing transaction: %w", err)
			}
			txn = db.NewTransaction(true)
			count = 0
		}
	}

	if count > 0 {
		if err := txn.Commit(); err != nil {
			return fmt.Errorf("error committing final transaction: %w", err)
		}
	}

	return nil
}

func fetchDatafromSourceURLs(urls map[string]bool, db *badger.DB) {
	client := &http.Client{Timeout: 30 * time.Second}
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"

	for url := range urls {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Error creating request for URL %s: %v\n", url, err)
			continue
		}
		req.Header.Set("User-Agent", userAgent) // Set the User-Agent header

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error fetching URL %s: %v\n", url, err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			lines := processResponseBody(resp.Body)
			if err := storeToBadgerDB(db, lines, url); err != nil {
				fmt.Printf("Error storing content from %s to BadgerDB: %v\n", url, err)
			} else {
				fmt.Printf("Processed and stored content from %s\n", url)
			}
		} else {
			fmt.Printf("URL %s returned status code %d\n", url, resp.StatusCode)
			resp.Body.Close()
		}
	}
}

func fetchDataFromBadgerDB(db *badger.DB, writer io.Writer) error {
	return db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			key := item.Key()
			err := item.Value(func(val []byte) error {
				_, writeErr := fmt.Fprintf(writer, "%s\n", key)
				return writeErr
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func main() {
	filePath := "wordlist-sources.yaml"
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		fmt.Printf("Error opening BadgerDB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	var outputPath string

	var rootCmd = &cobra.Command{
		Use:   "fuzzgen",
		Short: "Wordlist Generator",
	}

	var subdomainsCmd = &cobra.Command{
		Use:   "fuzzgen",
		Short: "Wordlist Generator",
		Run: func(cmd *cobra.Command, args []string) {
			// parse the sources file
			subdomainSourceURLs, err := parseSourcesYAML(filePath)
			if err != nil {
				fmt.Printf("Error parsing YAML file: %v\n", err)
				os.Exit(1)
			}

			// validate the urls in the sources file
			fmt.Printf("Checking %d subdomains sources...\n", len(subdomainSourceURLs))
			validateSourceURLs(subdomainSourceURLs)
			fmt.Println("Valid subdomain sources:")
			for url := range subdomainSourceURLs {
				fmt.Println(url)
			}

			// fetch data from the sources and store in database
			fetchDatafromSourceURLs(subdomainSourceURLs, db)

			// write output to file
			var writer io.Writer = os.Stdout
			if outputPath != "" {
				file, err := os.Create(outputPath)
				if err != nil {
					fmt.Printf("Error creating output file: %v\n", err)
					os.Exit(1)
				}
				defer file.Close()
				writer = file
			}

			if err := fetchDataFromBadgerDB(db, writer); err != nil {
				fmt.Printf("Error fetching data from BadgerDB: %v\n", err)
			}
		},
	}

	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Path to output file for storing results")
	rootCmd.AddCommand(subdomainsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
