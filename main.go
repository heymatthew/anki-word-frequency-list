package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"

	"golang.org/x/net/dict"
)

const (
	wordFrequencyList = "https://www.wordfrequency.info/samples/lemmas_60k.txt"
)

var (
	header       = regexp.MustCompile("^rank")
	cellSplitter = regexp.MustCompile("\t")
)

func main() {
	dictClient, err := connectDict("dict.org")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Dict Connect Error: %v", err)
		os.Exit(1)
	}
	defer dictClient.Close()

	books, err := englishDicts(dictClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "English Dicts Error: %v", err)
		os.Exit(1)
	}

	frequencyList, err := getFrequencyList()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Frequency List Error %v", err)
		os.Exit(1)
	}

	var lookupErrors int
	for i := 0; i < 10; i++ { // FIXME prefer len(frequencyList)
		word := frequencyList[i]
		for _, book := range books {
			definition, err := dictClient.Define(book.Name, word)
			if err != nil {
				lookupErrors++
				if lookupErrors < 100 {
					continue
				}
				fmt.Fprintf(os.Stderr, "More than 100 lookup failures: %v", err)
				os.Exit(1)
			}
			for _, meaning := range definition {
				fmt.Println(meaning.Word)
				fmt.Println(string(meaning.Text))

				// FIXME write out to import format for Anki
				// see https://apps.ankiweb.net/
			}
		}
	}
}

func englishDicts(dictClient *dict.Client) ([]dict.Dict, error) {
	dictTypes, err := dictClient.Dicts()
	if err != nil {
		return nil, fmt.Errorf("Dicts lookup Error: %v")
	}

	var englishDicts []dict.Dict
	for _, d := range dictTypes {
		switch d.Name {
		case "english", "jargon", "gcide":
			englishDicts = append(englishDicts, d)
		}
	}

	return englishDicts, nil
}

func connectDict(dictionaryUrl string) (*dict.Client, error) {
	ips, err := net.LookupIP(dictionaryUrl)
	if err != nil {
		return nil, fmt.Errorf("DNS Lookup Error: %v", err)
	}

	connectString := fmt.Sprintf("%s:dict", ips[0])
	d, err := dict.Dial("tcp", connectString)
	if err != nil {
		return nil, fmt.Errorf("Dial error: %v", err)
	}

	return d, nil
}

func getFrequencyList() ([]string, error) {
	resp, err := http.Get("https://www.wordfrequency.info/samples/lemmas_60k.txt")
	if err != nil {
		return nil, fmt.Errorf("Get Error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Status error: %v", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)

	var process bool
	var words []string
	for scanner.Scan() {
		line := scanner.Text()
		if header.MatchString(line) {
			process = true
			continue
		}
		if !process || len(line) == 0 {
			continue
		}

		cells := cellSplitter.Split(line, 3)
		words = append(words, cells[1])
	}

	return words, nil
}
