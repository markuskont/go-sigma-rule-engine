package cmd

import (
	"fmt"

	"github.com/markuskont/go-sigma-rule-engine/pkg/sigma"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// sigmaCmd represents the sigma command
var sigmaCmd = &cobra.Command{
	Use:   "sigma",
	Short: "",
	Long:  ``,
	Run:   entrypoint,
}

func entrypoint(cmd *cobra.Command, args []string) {
	var err error
	r, err := sigma.NewRuleset(
		&sigma.Config{
			Direcotries: viper.GetStringSlice("sigma.rules.dir"),
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(r)
	panic("BREAK")

	/*
		rules := make([]*sigma.RawRule, 0)
		if err = filepath.Walk(dir, func(
			path string,
			info os.FileInfo,
			err error,
		) error {
			if !info.IsDir() && strings.HasSuffix(path, "yml") {
				var s *sigma.RawRule
				data, err := ioutil.ReadFile(path) // just pass the file name
				if err != nil {
					log.WithFields(log.Fields{
						"file": path,
					}).Warn(err)
					return nil
				}
				if bytes.Contains(data, []byte("---")) {
					log.WithFields(log.Fields{
						"file": path,
					}).Warn("Multi-part yaml with key collisions. Thank you pythonistas.")
					return nil
				}
				if err := yaml.Unmarshal([]byte(data), &s); err != nil {
					log.WithFields(log.Fields{
						"file": path,
					}).Warn(err)
					return nil
				}
				if s.Detection == nil {
					log.WithFields(log.Fields{
						"title": s.Title,
						"file":  path,
					}).Warn("missing detection map, check rule")
					return nil
				}
				if _, err := s.Condition(); err != nil {
					log.WithFields(log.Fields{
						"title":     s.Title,
						"file":      path,
						"detection": s.Detection,
					}).Warnf("%s, check rule", err)
					return nil
				}
				s.File = path
				rules = append(rules, s)
			}
			return err
		}); err != nil {
			log.Fatal(err)
		}
		log.Infof("Got %d rules from %s", len(rules), dir)
		var good, bad, unhandled, unsupported int
		for _, rule := range rules {
			contextLogger := log.WithFields(log.Fields{
				"file": rule.File,
			})
			contextLogger.Trace("parsing rule")
			if tree, err := condition.Parse(rule.Detection); err != nil {
				switch err.(type) {
				case sigma.ErrUnsupportedToken, sigma.ErrIncompleteDetection:
					unsupported++
					contextLogger.Warn(err)
				case sigma.ErrWip:
					unsupported++
					contextLogger.Debugf(">>>>%s", err)
				default:
					bad++
					contextLogger.Error(err)
				}
			} else if tree != nil {
				contextLogger.Info("Seemed to parse fine")
				good++
			} else {
				unhandled++
			}
		}
		contextLogger := log.WithFields(log.Fields{
			"ok":          good,
			"errors":      bad,
			"unhandled":   unhandled,
			"unsupported": unsupported,
		})
		contextLogger.Info("Done")
	*/
}

func init() {
	rootCmd.AddCommand(sigmaCmd)

	sigmaCmd.PersistentFlags().StringSlice("sigma-rules-dir", []string{}, "Directories that contains sigma rules.")
	viper.BindPFlag("sigma.rules.dir", sigmaCmd.PersistentFlags().Lookup("sigma-rules-dir"))
}
