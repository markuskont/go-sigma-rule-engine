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
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/markuskont/go-dispatch"
	"github.com/markuskont/go-sigma-rule-engine/pkg/sigma/v2"
	"github.com/prometheus/common/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "A reference utility for matching sigma rules on event stream",
	Long: `Run command reads events from stdin, thus any stream could be piped into the command.
	For example:

	zcat ~/Logs/windows.json.gz | go-sigma-rule-engine run
	`,
	Run: run,
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func copyBytes(in []byte) []byte {
	tx := make([]byte, len(in))
	for i, b := range in {
		tx[i] = b
	}
	return tx
}

func scanLines(input io.Reader, ctx context.Context) <-chan []byte {
	tx := make(chan []byte, 1)
	go func(ctx context.Context) {
		defer close(tx)
		var count uint64
		scanner := bufio.NewScanner(input)
		tick := time.NewTicker(1 * time.Second)
		start := time.Now()
		for scanner.Scan() {
			select {
			case <-ctx.Done():
			case <-tick.C:
				logrus.Tracef("scanner got %d lines %.2f eps",
					count, float64(count)/float64(time.Since(start).Seconds()))
			case tx <- copyBytes(scanner.Bytes()):
				count++
			}
		}
		if err := scanner.Err(); err != nil {
			logrus.Fatal(err)
		}
	}(ctx)
	return tx
}

func open(path string) (io.ReadCloser, error) {
	var (
		file *os.File
		err  error
	)
	if file, err = os.Open(path); err != nil {
		return nil, err
	}
	if strings.HasSuffix(path, "gz") {
		return gzip.NewReader(file)
	}
	return file, nil
}

func run(cmd *cobra.Command, args []string) {
	var input io.ReadCloser
	var err error
	if infile := viper.GetString("sigma.input"); infile != "" {
		input, err = open(infile)
		if err != nil {
			log.Fatal(err)
		}
		defer input.Close()
	} else {
		input = os.Stdin
	}

	lines := scanLines(input, context.TODO())
	events := make(chan sigma.Event, 0)

	if err := dispatch.Run(dispatch.Config{
		Async:   true,
		Workers: viper.GetInt("decode.workers"),
		FeederFunc: func(tasks chan<- dispatch.Task, stop <-chan struct{}) {
			var wg sync.WaitGroup
			for i := 0; i < viper.GetInt("decode.workers"); i++ {
				wg.Add(1)
				tasks <- func(id, count int, ctx context.Context) error {
					defer wg.Done()
					for l := range lines {
						var d sigma.DynamicMap
						if err := json.Unmarshal(l, &d); err != nil {
							logrus.Fatal(err)
						}
						events <- d
					}
					return nil
				}
			}
			wg.Wait()
			close(events)
		},
		ErrFunc: func(err error) bool {
			return true
		},
	}); err != nil {
		logrus.Fatal(err)
	}
	if err := dispatch.Run(dispatch.Config{
		Async:   false,
		Workers: viper.GetInt("sigma.workers"),
		FeederFunc: func(tasks chan<- dispatch.Task, stop <-chan struct{}) {
			for i := 0; i < viper.GetInt("sigma.workers"); i++ {
				tasks <- func(id, count int, ctx context.Context) error {
					// ruleset is not thread safe, each worker needs a distinct copy
					ruleset, err := sigma.NewRuleset(sigma.Config{
						Directory: viper.GetStringSlice("rules.dir"),
					})
					if err != nil {
						return err
					}
					logrus.Debugf("Worker %d Found %d files, %d ok, %d failed, %d unsupported",
						id, ruleset.Total, ruleset.Ok, ruleset.Failed, ruleset.Unsupported)
					for e := range events {
						if result, match := ruleset.EvalAll(e); match {
							fmt.Printf("MATCH: %d rules", len(result))
						}
					}
					return nil
				}
			}
		},
		ErrFunc: func(err error) bool {
			return false
		},
	}); err != nil {
		logrus.Fatal(err)
	}
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().Int("decode-workers", 4,
		`Number of workers for decoding JSON events.`)
	viper.BindPFlag("decode.workers",
		runCmd.PersistentFlags().Lookup("decode-workers"))

	runCmd.PersistentFlags().Int("sigma-workers", 4,
		`Number of workers for sigma matching.`)
	viper.BindPFlag("sigma.workers",
		runCmd.PersistentFlags().Lookup("sigma-workers"))

	runCmd.PersistentFlags().String("sigma-input", "",
		`Input log file.`)
	viper.BindPFlag("sigma.input",
		runCmd.PersistentFlags().Lookup("sigma-input"))
}
