package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/markuskont/datamodels"
	"github.com/markuskont/go-sigma-rule-engine"
)

var (
	flagRuleSetPath = flag.String("path-ruleset", "", "Root folders for Sigma rules. Semicolon delimits paths.")
)

func main() {
	flag.Parse()
	if *flagRuleSetPath == "" {
		log.Fatal("ruleset path not configured")
	}
	ruleset, err := sigma.NewRuleset(sigma.Config{
		Directory:       strings.Split(*flagRuleSetPath, ";"),
		NoCollapseWS:    false,
		FailOnRuleParse: false,
		FailOnYamlParse: false,
	})
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(bufio.NewReader(os.Stdin))
	output := os.Stdout
loop:
	for scanner.Scan() {
		var obj datamodels.Map
		if err := json.Unmarshal(scanner.Bytes(), &obj); err != nil {
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
}
