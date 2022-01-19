package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

type conf struct {
	NodeIP             string  `yaml:"NodeIP"`
	NodePort           string  `yaml:"NodePort"`
	NodeUser           string  `yaml:"NodeUser"`
	NodePass           string  `yaml:"NodePass"`
	WalletsToMonitor   string  `yaml:"WalletsToMonitor"`
	ServerPort         string  `yaml:"ServerPort"`
	TelegramUserId     string  `yaml:"TelegramUserId"`
	MinerLateTime      float64 `yaml:"MinerLateTime"`
	AutoRefreshSeconds int     `yaml:"AutoRefreshSeconds"`
}

func (c *conf) getConf() *conf {

	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}
