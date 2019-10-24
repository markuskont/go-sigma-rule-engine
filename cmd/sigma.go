package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/ccdcoe/go-peek/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// sigmaCmd represents the sigma command
var sigmaCmd = &cobra.Command{
	Use:   "sigma",
	Short: "",
	Long:  ``,
	Run:   entrypoint,
}

type RawRule struct {
	// Our custom fields
	// Unique identifier that will be attached to positive match
	ID int `yaml:"id" json:"id"`
	// Detection logic type
	// Is it simple string match or more complex correlation

	// https://github.com/Neo23x0/sigma/wiki/Specification
	Title       string `yaml:"title" json:"title"`
	Status      string `yaml:"status" json:"status"`
	Description string `yaml:"description" json:"description"`
	Author      string `yaml:"author" json:"author"`
	// A list of URL-s to external sources
	References []string `yaml:"references" json:"references"`
	Logsource  struct {
		Product    string `yaml:"product" json:"product"`
		Category   string `yaml:"category" json:"category"`
		Service    string `yaml:"service" json:"service"`
		Definition string `yaml:"definition" json:"definition"`
	} `yaml:"logsource" json:"logsource"`

	Detection map[string]interface{} `yaml:"detection" json:"detection"`

	Fields         interface{} `yaml:"fields" json:"fields"`
	Falsepositives interface{} `yaml:"falsepositives" json:"falsepositives"`
	Level          interface{} `yaml:"level" json:"level"`
	Tags           []string    `yaml:"tags" json:"tags"`
}

func entrypoint(cmd *cobra.Command, args []string) {
	var err error
	var dir string
	if dir = viper.GetString("sigma.rules.dir"); dir == "" {
		log.Fatal("Missing rule directory, see --help")
	}
	if dir, err = utils.ExpandHome(dir); err != nil {
		log.Fatal(err)
	}
	rules := make([]*RawRule, 0)
	if err = filepath.Walk(dir, func(
		path string,
		info os.FileInfo,
		err error,
	) error {
		if !info.IsDir() && strings.HasSuffix(path, "yml") {
			var s RawRule
			data, err := ioutil.ReadFile(path) // just pass the file name
			if err != nil {
				log.WithFields(log.Fields{
					"file": path,
				}).Error(err)
				return nil
			}
			if err := yaml.Unmarshal([]byte(data), &s); err != nil {
				log.WithFields(log.Fields{
					"file": path,
				}).Error(err)
				return nil
			}
			if s.Detection == nil {
				log.WithFields(log.Fields{
					"title":     s.Title,
					"file":      path,
					"detection": s.Detection,
				}).Error("missing detection map, check rule")
				return nil
			}
			rules = append(rules, &s)
		}
		return err
	}); err != nil {
		log.Fatal(err)
	}
	log.Infof("Got %d rules from %s", len(rules), dir)
}

func init() {
	rootCmd.AddCommand(sigmaCmd)

	sigmaCmd.PersistentFlags().String("sigma-rules-dir", "", "Directory that contains sigma rules.")
	viper.BindPFlag("sigma.rules.dir", sigmaCmd.PersistentFlags().Lookup("sigma-rules-dir"))
}
