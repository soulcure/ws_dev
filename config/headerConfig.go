package config

import (
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type HeaderConfig struct {
	HeaderInfo HeaderInfo `yaml:"header"`
}

//Header配置
type HeaderInfo struct {
	Platform     string `yaml:"platform"`
	Version      string `yaml:"version"`
	Channel      string `yaml:"channel"`
	Device_id    string `yaml:"device_id"`
	Build_number string `yaml:"build_number"`
}

var (
	HeaderCfg HeaderConfig
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

	if err := yaml.Unmarshal(buf, &HeaderCfg); err != nil {
		logrus.Panic("db conf yaml Unmarshal error ")
	}

}
