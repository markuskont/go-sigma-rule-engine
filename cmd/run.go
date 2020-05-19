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

type stats struct {
	count int
	start time.Time
}

func (s stats) since() float64 {
	return time.Since(s.start).Seconds()
}

func (s stats) eps() float64 {
	return float64(s.count) / s.since()
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func copyBytes(in []byte) []byte {
	tx := make([]byte, len(in))
	for i, b := range in {
		tx[i] = b
	}
	return tx
}

func scanLines(input io.Reader, ctx context.Context, logFn func(stats)) <-chan []byte {
	tx := make(chan []byte, 1)
	go func(ctx context.Context) {
		defer close(tx)
		scanner := bufio.NewScanner(input)
		tick := time.NewTicker(viper.GetDuration("sigma.log.interval"))
		s := &stats{start: time.Now()}
	loop:
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				break loop
			case <-tick.C:
				if logFn != nil {
					logFn(*s)
				}
			case tx <- copyBytes(scanner.Bytes()):
				s.count++
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

	ctx := context.Background()
	timeout, cancel := context.WithTimeout(ctx,
		viper.GetDuration("sigma.consumer.timeout.value"))
	defer cancel()

	lines := scanLines(input, func() context.Context {
		if viper.GetBool("sigma.consumer.timeout.enable") {
			return timeout
		}
		return ctx
	}(), func(s stats) {
		logrus.Tracef("scanner got %d lines %.2f eps",
			s.count, s.eps())
	})
	matchDisable := viper.GetBool("sigma.disable.match")

	if err := dispatch.Run(dispatch.Config{
		Async:   false,
		Workers: viper.GetInt("sigma.workers"),
		FeederFunc: func(tasks chan<- dispatch.Task, stop <-chan struct{}) {
			var wg sync.WaitGroup
			for i := 0; i < viper.GetInt("sigma.workers"); i++ {
				wg.Add(1)
				tasks <- func(id, count int, ctx context.Context) error {
					defer wg.Done()
					ruleset, err := sigma.NewRuleset(sigma.Config{
						Directory: viper.GetStringSlice("rules.dir"),
					})
					if err != nil {
						return err
					}
					logrus.Debugf("Worker %d Found %d files, %d ok, %d failed, %d unsupported",
						id, ruleset.Total, ruleset.Ok, ruleset.Failed, ruleset.Unsupported)

				loop:
					for l := range lines {
						var d sigma.DynamicMap
						if err := json.Unmarshal(l, &d); err != nil {
							logrus.Fatal(err)
						}
						if matchDisable {
							continue loop
						}
						if result, match := ruleset.EvalAll(d); match {
							fmt.Printf("MATCH: %d rules", len(result))
						}
					}
					return nil
				}
			}
			wg.Wait()
		},
		ErrFunc: func(err error) bool {
			return true
		},
	}); err != nil {
		logrus.Fatal(err)
	}
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().Int("sigma-workers", 4,
		`Number of workers for sigma matching.`)
	viper.BindPFlag("sigma.workers",
		runCmd.PersistentFlags().Lookup("sigma-workers"))

	runCmd.PersistentFlags().String("sigma-input", "",
		`Input log file.`)
	viper.BindPFlag("sigma.input",
		runCmd.PersistentFlags().Lookup("sigma-input"))

	runCmd.PersistentFlags().Bool("sigma-disable-match", false,
		`Skips pattern matching. For measuring JSON decode performance.`)
	viper.BindPFlag("sigma.disable.match",
		runCmd.PersistentFlags().Lookup("sigma-disable-match"))

	runCmd.PersistentFlags().Bool("sigma-consumer-timeout-enable", false,
		`Enable timeout for consumer. For testing.`)
	viper.BindPFlag("sigma.consumer.timeout.enable",
		runCmd.PersistentFlags().Lookup("sigma-consumer-timeout-enable"))

	runCmd.PersistentFlags().Duration("sigma-consumer-timeout-value", 10*time.Second,
		`Duration value for consumer timeout if enabled.`)
	viper.BindPFlag("sigma.consumer.timeout.value",
		runCmd.PersistentFlags().Lookup("sigma-consumer-timeout-value"))

	runCmd.PersistentFlags().Duration("sigma-log-interval", 1*time.Second,
		`Interval between stats logging.`)
	viper.BindPFlag("sigma.log.interval",
		runCmd.PersistentFlags().Lookup("sigma-log-interval"))
}
