package main

import (
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type conf struct {
	AddrsToMonitor     string  `yaml:"AddrsToMonitor"`
	ServerPort         string  `yaml:"ServerPort"`
	TelegramUserId     string  `yaml:"TelegramUserId"`
	MinerLateTime      float64 `yaml:"MinerLateTime"`
	AutoRefreshSeconds int     `yaml:"AutoRefreshSeconds"`
	DailyStatDays      int     `yaml:"DailyStatDays"`
	QuietMode          bool    `yaml:"QuietMode"`
	AuthToken          string  `yaml:"AuthToken"`
}

func (myConfig *conf) getConf() *conf {
	myConfigFile := "config.yaml"
	if _, err := os.Stat("myconfig.yaml"); err == nil {
		myConfigFile = "myconfig.yaml"
	}

	yamlFile, err := ioutil.ReadFile(myConfigFile)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, myConfig)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return myConfig
}
