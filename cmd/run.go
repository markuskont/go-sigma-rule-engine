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
	"context"
	"io"
	"os"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/markuskont/go-dispatch"
	"github.com/markuskont/go-sigma-rule-engine/pkg/sigma/v2"
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

func run(cmd *cobra.Command, args []string) {
	lines := scanLines(os.Stdin, context.TODO())
	if err := dispatch.Run(dispatch.Config{
		Async:   false,
		Workers: viper.GetInt("decode.workers"),
		FeederFunc: func(tasks chan<- dispatch.Task, stop <-chan struct{}) {
			for i := 0; i < viper.GetInt("decode.workers"); i++ {
				tasks <- func(id, count int, ctx context.Context) error {
					for l := range lines {
						var d sigma.DynamicMap
						if err := json.Unmarshal(l, &d); err != nil {
							logrus.Fatal(err)
						}
					}
					return nil
				}
			}
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

	rootCmd.PersistentFlags().Int("decode-workers", 4,
		`Number of workers for decoding JSON events.`)
	viper.BindPFlag("decode.workers",
		rootCmd.PersistentFlags().Lookup("decode-workers"))
}
