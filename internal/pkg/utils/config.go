package utils

import (
	"encoding/json"
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
	Host string `json:"host"`
	Port int    `json:"port"`
}

func ReadYamlConfig(filename string, conf ConfigValidator) error {
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

func ReadJsonConfig(filename string, conf ConfigValidator) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, conf.(interface{}))
	if err != nil {
		return err
	}

	return conf.Validate()
}

func (g *GraphiteSettings) Validate() error {
	if g != nil {
		if g.Host == "" {
			return errors.New("empty host string")
		}

		if g.Port < 1 || g.Port > 65535 {
			return errors.New(fmt.Sprintf("wrong graphite port: %d", g.Port))
		}
	}

	return nil
}
