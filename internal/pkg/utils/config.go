package utils

import (
	"errors"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type ConfigValidator interface {
	Validate() error
}

type DbSettings struct {
	Host     string `yaml:"host"`
	Port     uint16 `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type GraphiteSettings struct {
	Protocol string `yaml:"protocol"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Prefix   string `yaml:"prefix"`
}

func ReadConfig(filename string, conf ConfigValidator) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, conf.(interface{}))
	if err != nil {
		return err
	}

	return conf.Validate()
}

func (g *GraphiteSettings) Validate() error {
	if g != nil {
		switch g.Protocol {
		case "nop", "tcp", "udp":
		default:
			return errors.New(fmt.Sprintf("unknown metrics protocol: %s", g.Protocol))
		}

		if g.Port < 1 || g.Port > 65535 {
			return errors.New(fmt.Sprintf("wrong graphite port: %d", g.Port))
		}
	}

	return nil
}
