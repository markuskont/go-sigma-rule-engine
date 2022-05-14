package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/markuskont/datamodels"
	"github.com/markuskont/go-sigma-rule-engine"
)

var (
	flagRuleSetPath = flag.String("path-ruleset", "", "Root folders for Sigma rules. Semicolon delimits paths.")
	flagWorkers     = flag.Int("workers", 4, "Number of async workers")
)

func main() {
	flag.Parse()
	if *flagRuleSetPath == "" {
		log.Fatal("ruleset path not configured")
	}
	if *flagWorkers <= 0 {
		log.Fatal("invalid worker count")
	}

	// ruleset setup
	ruleset, err := sigma.NewRuleset(sigma.Config{
		Directory:       strings.Split(*flagRuleSetPath, ";"),
		NoCollapseWS:    false,
		FailOnRuleParse: false,
		FailOnYamlParse: false,
	})
	if err != nil {
		log.Fatal(err)
	}

	// syncing setup
	var wg sync.WaitGroup
	defer wg.Wait()
	ch := make(chan []byte, *flagWorkers)

	// workers setup
	for i := 0; i < *flagWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			output := os.Stdout
		loop:
			for data := range ch {
				var obj datamodels.Map
				if err := json.Unmarshal(data, &obj); err != nil {
					log.Println(err)
					continue loop
				}
				if results, ok := ruleset.EvalAll(obj); ok && len(results) > 0 {
					obj["sigma_results"] = results
					encoded, err := json.Marshal(obj)
					if err != nil {
						log.Println(err)
						continue loop
					}
					output.Write(append(encoded, []byte("\n")...))
				}
			}
		}()
	}

	// scanner setup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ch)
		scanner := bufio.NewScanner(bufio.NewReader(os.Stdin))
		for scanner.Scan() {
			// need to copy the bytes as scanner.Bytes is modified in place
			cpy := make([]byte, len(scanner.Bytes()))
			copy(cpy, scanner.Bytes())
			ch <- cpy
		}
	}()
}
