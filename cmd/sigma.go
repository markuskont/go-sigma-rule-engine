package cmd

import (
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
}

func init() {
	rootCmd.AddCommand(sigmaCmd)

	sigmaCmd.PersistentFlags().StringSlice("sigma-rules-dir", []string{}, "Directories that contains sigma rules.")
	viper.BindPFlag("sigma.rules.dir", sigmaCmd.PersistentFlags().Lookup("sigma-rules-dir"))
}
