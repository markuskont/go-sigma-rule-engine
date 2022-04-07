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
package cmd

import (
	"github.com/markuskont/go-sigma-rule-engine/pkg/sigma/v2"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type counts struct {
	ok, fail, unsupported int
}

// parseCmd represents the parse command
var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse a ruleset for testing",
	Long:  `Recursively parses a sigma ruleset from filesystem and provides detailed feedback to the user about rule support.`,
	Run:   parse,
}

func parse(cmd *cobra.Command, args []string) {
	files, err := sigma.NewRuleFileList(viper.GetStringSlice("rules.dir"))
	if err != nil {
		logrus.Fatal(err)
	}
	for _, f := range files {
		logrus.Info(f)
	}
	logrus.Info("Parsing rule yaml files")
	rules, err := sigma.NewRuleList(files, true, false)
	if err != nil {
		switch err.(type) {
		case sigma.ErrBulkParseYaml:
			logrus.Error(err)
		default:
			logrus.Fatal(err)
		}
	}
	logrus.Infof("Got %d rules from yaml", len(rules))
	logrus.Info("Parsing rules into AST")
	c := &counts{}
loop:
	for _, raw := range rules {
		logrus.Trace(raw.Path)
		if raw.Multipart {
			c.unsupported++
			continue loop
		}
		_, err := sigma.NewTree(raw)
		if err != nil {
			switch err.(type) {
			case sigma.ErrUnsupportedToken:
				c.unsupported++
				logrus.Warnf("%s: %s", err, raw.Path)
			default:
				c.fail++
				logrus.Errorf("%s", err)
			}
		} else {
			logrus.Infof("%s: ok", raw.Path)
			c.ok++
		}
	}
	logrus.Infof("OK: %d; FAIL: %d; UNSUPPORTED: %d", c.ok, c.fail, c.unsupported)
}

func init() {
	rootCmd.AddCommand(parseCmd)
}
