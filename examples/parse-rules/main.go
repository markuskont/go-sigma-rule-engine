/*
Copyright © 2020 Markus Kont alias013@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"flag"
	"log"
	"strings"

	"github.com/markuskont/go-sigma-rule-engine"
)

type counts struct {
	ok, fail, unsupported int
}

var (
	flagRuleDir = flag.String("rules-dir", "", "Directories containing rules. Multiple can be defined with semicolon as separator.")
)

func main() {
	flag.Parse()
	files, err := sigma.NewRuleFileList(strings.Split(*flagRuleDir, ";"))
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		log.Println(f)
	}
	log.Println("Parsing rule yaml files")
	rules, err := sigma.NewRuleList(files, true, false, nil)
	if err != nil {
		switch err.(type) {
		case sigma.ErrBulkParseYaml:
			log.Println(err)
		default:
			log.Fatal(err)
		}
	}
	log.Printf("Got %d rules from yaml\n", len(rules))
	log.Println("Parsing rules into AST")
	c := &counts{}
loop:
	for _, raw := range rules {
		log.Print(raw.Path)
		if raw.Multipart {
			c.unsupported++
			continue loop
		}
		_, err := sigma.NewTree(raw)
		if err != nil {
			switch err.(type) {
			case sigma.ErrUnsupportedToken:
				c.unsupported++
				log.Printf("%s: %s\n", err, raw.Path)
			default:
				c.fail++
				log.Printf("%s\n", err)
			}
		} else {
			log.Printf("%s: ok\n", raw.Path)
			c.ok++
		}
	}
	log.Printf("OK: %d; FAIL: %d; UNSUPPORTED: %d\n", c.ok, c.fail, c.unsupported)
}
