package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/markuskont/datamodels"
	"github.com/markuskont/go-sigma-rule-engine"
)

var (
	flagRuleSetPath = flag.String("path-ruleset", "./windows/", "Root folders for Sigma rules. Semicolon delimits paths.")
)

func saveJSONToFile(filename string, data []interface{}) error {
	var jsonData []byte
	for _, d := range data {
		dJSON, err := json.Marshal(d)
		if err != nil {
			return err
		}
		jsonData = append(jsonData, dJSON...)
		jsonData = append(jsonData, '\n')
	}
	return ioutil.WriteFile(filename, jsonData, 0644)
}
func main() {

	log.Println("start job")
	flag.Parse()

	if *flagRuleSetPath == "" {
		log.Fatal("ruleset path not configured")
	}

	ruleset, err := sigma.NewRuleset(sigma.Config{
		Directory:       strings.Split(*flagRuleSetPath, ";"),
		NoCollapseWS:    false,
		FailOnRuleParse: false,
		FailOnYamlParse: false,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadFile("./data.json")

	if err != nil {
		log.Fatal(err)
	}

	var events []map[string]interface{}

	if err := json.Unmarshal([]byte(data), &events); err != nil {
		panic(err)
	}
	cnt := 0
	hit := 0
	sigmaResults := []interface{}{}
	for _, event := range events {

		jsonStr, err := json.Marshal(event)

		if err != nil {
			log.Println(err)
		}

		var obj datamodels.Map
		if err := json.Unmarshal(jsonStr, &obj); err != nil {
			log.Println(err)
		}

		if results, ok := ruleset.EvalAll(obj); ok && len(results) > 0 {
			obj["sigma_results"] = results
			if err != nil {
				log.Println(err)
			}
			sigmaResults = append(sigmaResults, obj)

			hit += 1
		}
		cnt += 1

	}
	log.Println("total dataset : ", cnt)
	log.Println("total hit rule : ", hit)
	file, err := os.Create("./mapping_results.json")
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(sigmaResults)
}
