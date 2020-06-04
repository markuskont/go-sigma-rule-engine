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
	"os/signal"
	"path"
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

func sumList(rx *list.List) int64 {
	if rx.Len() == 0 {
		return 0
	}
	var sum int64
	for e := rx.Front(); e != nil; e = e.Next() {
		sum += e.Value.(time.Duration).Nanoseconds()
	}
	return sum
}

type timeStats struct {
	ID int

	ruleCount int

	decode *list.List
	match  *list.List
}

func newTimeStats(id, ruleCount int) *timeStats {
	return &timeStats{
		ID:        id,
		ruleCount: ruleCount,
		decode:    list.New(),
		match:     list.New(),
	}
}

type stats struct {
	start time.Time

	timeStats *timeStats

	Timestamp           time.Time `json:"timestamp"`
	Count               int       `json:"count"`
	EPS                 float64   `json:"eps"`
	AvgDecodeNano       int64     `json:"avg_decode_nano"`
	AvgMatchRulesetNano int64     `json:"avg_match_ruleset_nano"`
	AvgMatchPerRuleNano int64     `json:"avg_match_per_rule_nano"`
	RuleCount           int       `json:"rule_count"`
	MatchResults        int       `json:"match_results"`

	k                int64
	totalDecodeNanos int64
	totalMatchNanos  int64
}

func newStats(id, ruleCount int) *stats {
	return &stats{
		start:     time.Now(),
		timeStats: newTimeStats(id, ruleCount),
	}
}

func (s *stats) now() *stats {
	s.Timestamp = time.Now()
	return s
}

func (s stats) since() float64 {
	return time.Since(s.start).Seconds()
}

func (s stats) eps() float64 {
	return float64(s.Count) / s.since()
}

func (s *stats) calculate() *stats {
	s.EPS = s.eps()
	if s.k > 0 {
		s.AvgDecodeNano = s.totalDecodeNanos / s.k
		s.AvgMatchRulesetNano = s.totalMatchNanos / s.k
	}
	if s.timeStats.ruleCount > 0 {
		s.RuleCount = s.timeStats.ruleCount
		s.AvgMatchPerRuleNano = s.AvgMatchRulesetNano / int64(s.timeStats.ruleCount)
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
	return fmt.Sprintf("%d,%.2f,%d,%d,%d,%d,%d",
		s.Count, s.EPS, s.AvgDecodeNano, s.AvgMatchRulesetNano,
		s.AvgMatchPerRuleNano, s.RuleCount, s.MatchResults)
}

func (s stats) header() string {
	return strings.Join([]string{
		"count", "eps", "avg_decode_nano", "avg_match_ruleset_nano",
		"avg_match_per_rule_nano", "rule_count", "match_results",
	}, ",")
}

func (s stats) json() (string, error) {
	b, err := json.Marshal(s.calculate())
	if err != nil {
		return string(b), err
	}
	return string(b), nil
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func copyBytes(in []byte) []byte {
	tx := make([]byte, len(in))
	for i, b := range in {
		tx[i] = b
	}
	return tx
}

type logMessage struct {
	Data    []byte
	Offset  int
	Results sigma.Results
}

func scanLines(input io.Reader, ctx context.Context, logFn func(int, int)) <-chan logMessage {
	tx := make(chan logMessage, 1)
	go func(ctx context.Context) {
		defer close(tx)
		scanner := bufio.NewScanner(input)
		tick := time.NewTicker(100 * time.Millisecond)
		var count, last int
	loop:
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				break loop
			case <-tick.C:
				if logFn != nil {
					logFn(count, count-last)
				}
				last = count
			case tx <- logMessage{Data: copyBytes(scanner.Bytes()), Offset: count}:
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
func logStats(
	ingestCh <-chan int,
	workerCh <-chan timeStats,
	resultCh <-chan sigma.Results,
	ctx context.Context,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	defer logrus.Info("Stats logger done")
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
	s := newStats(0, 0)

loop:
	for {
		select {
		case <-tick.C:
			logrus.Trace(s.now())

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
		case count, ok := <-ingestCh:
			if !ok {
				continue loop
			}
			s.increment(count)
		case s2, ok := <-workerCh:
			if !ok {
				continue loop
			}
			s.totalDecodeNanos += sumList(s2.decode)
			s.totalMatchNanos += sumList(s2.match)
			s.k += int64(s2.decode.Len())
			s.timeStats.ruleCount = s2.ruleCount
		case res, ok := <-resultCh:
			if !ok {
				continue loop
			}
			s.MatchResults += len(res)
		case <-ctx.Done():
			break loop
		}
	}
}

func run(cmd *cobra.Command, args []string) {
	var (
		input io.ReadCloser
		err   error
		wg    sync.WaitGroup
	)
	if infile := viper.GetString("sigma.input"); infile != "" {
		input, err = open(infile)
		if err != nil {
			log.Fatal(err)
		}
		defer input.Close()
	} else {
		input = os.Stdin
	}

	ctx, cancel1 := context.WithCancel(context.Background())
	timeout, cancel2 := context.WithTimeout(ctx,
		viper.GetDuration("sigma.consumer.timeout.value"))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		logrus.Info("Caught SIGINT, stopping consumer")
		if viper.GetBool("sigma.consumer.timeout.enable") {
			cancel2()
		} else {
			cancel1()
		}
	}()

	ingestStatCh := make(chan int, 0)
	workerStatCh := make(chan timeStats, viper.GetInt("sigma.workers"))
	resultCh := make(chan sigma.Results, viper.GetInt("sigma.workers"))

	lines := scanLines(input, func() context.Context {
		if viper.GetBool("sigma.consumer.timeout.enable") {
			logrus.Infof("Enabling consumer timeout after %s",
				viper.GetDuration("sigma.consumer.timeout.value").String())
			return timeout
		}
		return ctx
	}(), func(count, diff int) {
		ingestStatCh <- diff
	})
	wg.Add(1)
	logCtx, logCancel := context.WithCancel(context.Background())
	go logStats(ingestStatCh, workerStatCh, resultCh, logCtx, &wg)

	matchDisable := viper.GetBool("sigma.disable.match")
	if matchDisable {
		logrus.Println("Disabling match engine.")
	}

	eventCh, enableMatchLogging := func() (chan<- logMessage, bool) {
		if path := viper.GetString("sigma.emit.file"); path != "" {
			handle, err := os.Create(path)
			if err != nil {
				logrus.Fatal(err)
			}
			eCh := make(chan logMessage, viper.GetInt("sigma.workers"))
			wg.Add(1)
			go func(ch <-chan logMessage, handle io.WriteCloser) {
				defer handle.Close()
				defer wg.Done()
				defer logrus.Info("Emitter done and file handle flushed")
				for e := range eCh {
					handle.Write(e.Data)
				}
			}(eCh, handle)
			return eCh, true
		}
		return nil, false
	}()

	ruleProfileDir, ruleProfileEnabled := func() (string, bool) {
		if dir := viper.GetString("sigma.rule.profile.dir"); dir != "" {
			if stat, err := os.Stat(dir); err != nil {
				log.Fatalf("Profiling path %s error %s", dir, err)
			} else if !stat.IsDir() {
				logrus.Fatalf("Profiling path %s exists but is not a directory", dir)
			} else {
				return dir, true
			}
		}
		return "", false
	}()
	if ruleProfileEnabled {
		logrus.Infof("Enabling rule profiling to %s", ruleProfileDir)
	}
	// spawn logger routine here

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
						Profile:   ruleProfileEnabled,
					})
					if err != nil {
						return err
					}
					logrus.Debugf("Worker %d Found %d files, %d ok, %d failed, %d unsupported",
						id, ruleset.Total, ruleset.Ok, ruleset.Failed, ruleset.Unsupported)

					s := newTimeStats(id, ruleset.Ok)
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
							if err := json.Unmarshal(l.Data, &d); err != nil {
								logrus.Fatal(err)
							}
							s.decode.PushBack(time.Since(start))
							if matchDisable {
								continue loop
							}
							start = time.Now()
							if result, match := ruleset.EvalAll(d); match {
								logrus.Infof("MATCH at offset %d : %v \n", l.Offset, result)
								resultCh <- result
								if enableMatchLogging {
									l.Results = result
									eventCh <- l
								}
							}
							s.match.PushBack(time.Since(start))
						case <-report.C:
							if len(workerStatCh) == viper.GetInt("sigma.workers") {
								<-workerStatCh
							}
							workerStatCh <- *s
							s = newTimeStats(id, ruleset.Ok)
						}
					}
					// TODO - refactor
					if ruleProfileEnabled {
						logrus.Infof("Worker %d rule profiling enabled", id)
					loop2:
						for k, v := range ruleset.Measurements {
							logrus.Infof("Worker %d dumping rule %s with %d measurements", id, k, v.Len())
							p := path.Join(ruleProfileDir, fmt.Sprintf("worker-%d-rule-%s.json", id, k))
							handle, err := os.Create(p)
							if err != nil {
								logrus.Fatal(err)
							}
							for e := v.Front(); e != nil; e = e.Next() {
								select {
								case <-ctx.Done():
									break loop2
								default:
								}
								val := e.Value.(sigma.RuleProfileItem)
								b, err := json.Marshal(val)
								if err != nil {
									logrus.Fatal(err)
								}
								b = append(b, []byte("\n")...)
								handle.Write(b)
							}
							handle.Sync()
							handle.Close()
						}
					}
					return nil
				}
			}
			wg.Wait()
			if enableMatchLogging {
				close(eventCh)
			}
		},
		ErrFunc: func(err error) bool {
			return true
		},
	}); err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("All workers exited, waiting on loggers to finish")
	logCancel()
	wg.Wait()
	logrus.Info("Done")
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

	runCmd.PersistentFlags().String("sigma-emit-file", "",
		`Destination file for storing events that match a sigma rule.`)
	viper.BindPFlag("sigma.emit.file",
		runCmd.PersistentFlags().Lookup("sigma-emit-file"))

	runCmd.PersistentFlags().String("sigma-rule-profile-dir", "",
		`Destination directory for storing per rule profiling information.`)
	viper.BindPFlag("sigma.rule.profile.dir",
		runCmd.PersistentFlags().Lookup("sigma-rule-profile-dir"))
}
