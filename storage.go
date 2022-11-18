package main

import (
	"encoding/json"
	"log"
	"os"
	"path"
)

type Config struct {
	Selected         *Set
	Downloaded       map[string]string
	APIKey           string
	UserId           string
	DownloadLocation string
	APIEndpoint      string
}

type writedConfig struct {
	Selected         []string
	Downloaded       map[string]string
	APIKey           string
	UserId           string
	DownloadLocation string
	APIEndpoint      string
}

func getConfigFilePath() string {
	configFolder, err := os.UserConfigDir()
	checkError(err)

	return path.Join(configFolder, "jellyfindl.json")
}

func writeConfig(conf Config) {
	writedConf := writedConfig{
		conf.Selected.Values(),
		conf.Downloaded,
		conf.APIKey,
		conf.UserId,
		conf.DownloadLocation,
		conf.APIEndpoint,
	}
	b, err := json.Marshal(writedConf)
	if err != nil {
		panic(err)
	}

	os.WriteFile(getConfigFilePath(), b, os.ModePerm)
}

func getConfig() *Config {
	b, err := os.ReadFile(getConfigFilePath())
	conf := writedConfig{Selected: make([]string, 0)}
	if err != nil {
		if os.IsNotExist(err) {
			b = []byte("{}")
			os.WriteFile(getConfigFilePath(), b, os.ModePerm)
		}
	}

	err = json.Unmarshal(b, &conf)
	if err != nil {
		log.Panic(err)
	}

	selected := NewSet()
	selected.AddAll(conf.Selected)
	if conf.Downloaded == nil {
		conf.Downloaded = make(map[string]string)
	}

	return &Config{
		selected,
		conf.Downloaded,
		conf.APIKey,
		conf.UserId,
		conf.DownloadLocation,
		conf.APIEndpoint,
	}
}
