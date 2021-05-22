package config

import (
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

//WebSocket配置
type RuleConfig struct {
	Content string `yaml:"content"`
	Times   int    `yaml:"times"`
}

var (
	RuleCfg RuleConfig
)

func init() {
	f, openErr := os.Open("./fanbook.yml")
	if openErr != nil {
		logrus.Panic("failed to open config file:", openErr)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			logrus.Println("failed to close config file:", closeErr)
		}
	}()
	buf, readErr := ioutil.ReadAll(f)
	if readErr != nil {
		logrus.Println("failed to read config file:", readErr)
	}

	if err := yaml.Unmarshal(buf, &RuleCfg); err != nil {
		logrus.Panic("db conf yaml Unmarshal error ")
	}

}
