package conf

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Config struct {
	PeerName   string
	PeerAddr   string
	PeerPort   string
	PeerID     string
	Group      string
	Neighbor   string
	ConfigPath string
	NodePort   string
}

var conf Config

func GetConfig() Config {
	return conf
}

var DefaultConfigPath = "./conf.toml"

func InitConfig(path string) error {
	if path == "" {
		path = DefaultConfigPath
	}
	if _, err := os.Stat(path); err != nil {
		return errors.Wrap(err, "config file not exist")
	}
	viper.SetConfigFile(path)
	viper.SetConfigType("toml")
	if err := viper.ReadInConfig(); err != nil {
		return errors.Wrap(err, "can not load config file")
	}
	if err := viper.Unmarshal(&conf); err != nil {
		return errors.Wrap(err, "unmarshal config file error")
	}
	return nil
}
