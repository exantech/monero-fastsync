package utils

import (
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
