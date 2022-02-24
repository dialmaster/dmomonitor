package main

import (
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type conf struct {
	ServerPort         string  `yaml:"ServerPort"`
	MinerLateTime      float64 `yaml:"MinerLateTime"`
	AutoRefreshSeconds int     `yaml:"AutoRefreshSeconds"`
	DailyStatDays      int     `yaml:"DailyStatDays"`
	DBUser             string  `yaml:"DBUser"`
	DBPass             string  `yaml:"DBPass"`
	DBIP               string  `yaml:"DBIP"`
	DBPort             string  `yaml:"DBPort"`
	DBName             string  `yaml:"DBName"`
}

func (myConfig *conf) getConf() *conf {
	myConfigFile := "config.yaml"
	if _, err := os.Stat("myconfig.yaml"); err == nil {
		myConfigFile = "myconfig.yaml"
	}

	yamlFile, err := ioutil.ReadFile(myConfigFile)
	if err != nil {
		log.Fatalf("Error loading yaml config: #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, myConfig)
	if err != nil {
		log.Fatalf("Error unmarshalling yaml config: %v", err)
	}

	return myConfig
}
