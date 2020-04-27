package cmd

import (
	"github.com/markuskont/go-sigma-rule-engine/pkg/sigma/v2"
	"github.com/sirupsen/logrus"
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
	files, err := sigma.NewRuleFileList(viper.GetStringSlice("sigma.rules.dir"))
	if err != nil {
		logrus.Fatal(err)
	}
	for _, f := range files {
		logrus.Info(f)
	}
	rules, err := sigma.NewRuleList(files, true)
	if err != nil {
		switch err.(type) {
		case sigma.ErrBulkParseYaml:
			logrus.Error(err)
		default:
			logrus.Fatal(err)
		}
	}
	logrus.Infof("Got %d rules", len(rules))
	for _, rule := range rules {
		if val, ok := rule.Detection["condition"].(string); ok {
			logrus.Info(val)
		} else if rule.Multipart {
			logrus.Warnf("%s is multipart", rule.Path)
		} else {
			logrus.Errorf("%s missing condition or not string", rule.Path)
		}
	}
	/*
		r, err := sigma.NewRuleset(
			&sigma.Config{
				Directories: viper.GetStringSlice("sigma.rules.dir"),
			},
		)
		if err != nil {
			log.Fatal(err)
		}
		if len(r.Unsupported) > 0 {
			for _, rule := range r.Unsupported {
				log.Warn(rule)
			}
		}
		if len(r.Broken) > 0 {
			for _, rule := range r.Broken {
				log.Error(rule)
			}
		}

		contextLogger := log.WithFields(log.Fields{
			"ok":          r.Total,
			"errors":      len(r.Broken),
			"unsupported": len(r.Unsupported),
		})
		contextLogger.Info("Done")
	*/
}

func init() {
	rootCmd.AddCommand(sigmaCmd)

	sigmaCmd.PersistentFlags().StringSlice("sigma-rules-dir", []string{},
		"Directories that contains sigma rules.")
	viper.BindPFlag("sigma.rules.dir", sigmaCmd.PersistentFlags().Lookup("sigma-rules-dir"))
}
