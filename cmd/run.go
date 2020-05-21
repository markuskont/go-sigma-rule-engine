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
	"container/list"
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

func calculateAverageNanos(rx *list.List) int64 {
	if rx.Len() == 0 {
		return 0
	}
	var count, sum int64
	for e := rx.Front(); e != nil; e = e.Next() {
		count++
		sum += e.Value.(time.Duration).Nanoseconds()
	}
	return sum / count
}

type stats struct {
	start  time.Time
	decode *list.List
	match  *list.List

	Timestamp     time.Time `json:"timestamp"`
	Count         int       `json:"count"`
	EPS           float64   `json:"eps"`
	AvgDecodeNano int64     `json:"avg_decode_nano"`
	AvgMatchNano  int64     `json:"avg_match_nano"`
}

func newStats() *stats {
	return &stats{
		start:  time.Now(),
		match:  list.New(),
		decode: list.New(),
	}
}

func (s stats) since() float64 {
	return time.Since(s.start).Seconds()
}

func (s stats) eps() float64 {
	return float64(s.Count) / s.since()
}

func (s *stats) calculate() *stats {
	s.EPS = s.eps()
	if s.decode != nil {
		s.AvgDecodeNano = calculateAverageNanos(s.decode)
	}
	if s.match != nil {
		s.AvgMatchNano = calculateAverageNanos(s.match)
	}
	return s
}

func (s *stats) set(count int) *stats {
	s.Count = count
	return s
}

func (s *stats) increment(count int) *stats {
	s.Count += count
	return s
}
func (s stats) String() string {
	return fmt.Sprintf("scanner got %d lines %.2f eps", s.Count, s.eps())
}

func (s stats) csv() string {
	s.calculate()
	return fmt.Sprintf("%d,%.2f,%d,%d", s.Count, s.EPS, s.AvgDecodeNano, s.AvgMatchNano)
}

func (s stats) header() string {
	return strings.Join([]string{
		"count", "eps", "avg_decode_nano", "avg_match_nano",
	}, ",")
}

func (s stats) json() (string, error) {
	b, err := json.Marshal(s.calculate())
	if err != nil {
		return string(b), err
	}
	return string(b), nil
}

func (s *stats) calculateDecode(start time.Time) *stats {
	if s.decode != nil {
		s.decode.PushBack(time.Since(start))
	}
	return s
}

func (s *stats) calculateMatch(start time.Time) *stats {
	if s.match != nil {
		s.match.PushBack(time.Since(start))
	}
	return s
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func copyBytes(in []byte) []byte {
	tx := make([]byte, len(in))
	for i, b := range in {
		tx[i] = b
	}
	return tx
}

func scanLines(input io.Reader, ctx context.Context, logFn func(int)) <-chan []byte {
	tx := make(chan []byte, 1)
	go func(ctx context.Context) {
		defer close(tx)
		scanner := bufio.NewScanner(input)
		tick := time.NewTicker(100 * time.Millisecond)
		var count int
	loop:
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				break loop
			case <-tick.C:
				if logFn != nil {
					logFn(count)
				}
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

type statLogFmt int

const (
	statLogPlain statLogFmt = iota
	statLogCsv
	statLogJSON
)

// goroutine
func logStats(ingestCh <-chan int) {
	statFile, statFileEnabled := func() (io.WriteCloser, bool) {
		if path := viper.GetString("sigma.stats.file"); path != "" {
			handle, err := os.Create(path)
			if err != nil {
				logrus.Fatal(err)
			}
			return handle, true
		}
		return nil, false
	}()
	if statFileEnabled {
		defer statFile.Close()
	}

	format := func() statLogFmt {
		switch viper.GetString("sigma.stats.format") {
		case "csv":
			if statFileEnabled {
				fmt.Fprintln(statFile, stats{}.header())
			}
			return statLogCsv
		case "json":
			return statLogJSON
		default:
			return statLogPlain
		}
	}()

	tick := time.NewTicker(viper.GetDuration("sigma.stats.interval"))
	s := newStats()

loop:
	for {
		select {
		case <-tick.C:
			logrus.Trace(s)

			if !statFileEnabled {
				continue loop
			}
			fmt.Fprintln(statFile, func() string {
				switch format {
				case statLogCsv:
					return s.csv()
				case statLogJSON:
					j, err := s.json()
					if err != nil {
						logrus.Error(err)
					}
					return j
				default:
					return s.String()
				}
			}())
			//s = newStats()
		case count, ok := <-ingestCh:
			if !ok {
				break loop
			}
			s.set(count)
		}
	}
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

	ingestStatCh := make(chan int, 0)
	lines := scanLines(input, func() context.Context {
		if viper.GetBool("sigma.consumer.timeout.enable") {
			logrus.Infof("Enabling consumer timeout after %s",
				viper.GetDuration("sigma.consumer.timeout.value").String())
			return timeout
		}
		return ctx
	}(), func(count int) {
		ingestStatCh <- count
	})
	go logStats(ingestStatCh)

	matchDisable := viper.GetBool("sigma.disable.match")
	if matchDisable {
		logrus.Println("Disabling match engine.")
	}

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

					s := newStats()
					report := time.NewTicker(1 * time.Second)

				loop:
					for {
						select {
						case l, ok := <-lines:
							if !ok {
								break loop
							}
							start := time.Now()
							var d sigma.DynamicMap
							if err := json.Unmarshal(l, &d); err != nil {
								logrus.Fatal(err)
							}
							s.calculateDecode(start)
							if matchDisable {
								continue loop
							}
							start = time.Now()
							if _, match := ruleset.EvalAll(d); match {
								//fmt.Printf("MATCH: %d rules\n", len(result))
							}
							s.calculateMatch(start)
						case <-report.C:
							s = newStats()
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

	runCmd.PersistentFlags().Duration("sigma-stats-interval", 1*time.Second,
		`Interval between stats logging.`)
	viper.BindPFlag("sigma.stats.interval",
		runCmd.PersistentFlags().Lookup("sigma-stats-interval"))

	runCmd.PersistentFlags().String("sigma-stats-file", "",
		`Log file for stats.`)
	viper.BindPFlag("sigma.stats.file",
		runCmd.PersistentFlags().Lookup("sigma-stats-file"))

	runCmd.PersistentFlags().String("sigma-stats-format", "human",
		`Log format for performance statistics. Supported values are:
		human - unstructured plaintext
		json - key and value JSON messages
		csv - comma separated values`)
	viper.BindPFlag("sigma.stats.format",
		runCmd.PersistentFlags().Lookup("sigma-stats-format"))
}
