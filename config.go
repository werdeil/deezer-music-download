package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func getConfig() (configuration, error) {
	var err error
	var config configuration

	configDir := os.Getenv("XDG_CONFIG_HOME")
	if len(configDir) == 0 {
		homedir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		configDir = filepath.Join(homedir, ".config")
	}
	configPath := filepath.Join(configDir, "deezer-music-download", "config.toml")

	_, err = toml.DecodeFile(configPath, &config)
	if errors.Is(err, os.ErrNotExist) {
		return configuration{}, fmt.Errorf("config file not found at %s (copy example_config.toml there and fill in the values)", configPath)
	}
	if err != nil {
		return configuration{}, err
	}
	if len(config.Arl) == 0 {
		return configuration{}, errors.New("please provide a value for the 'arl' field in the config file")
	}
	// license_token is optional here; in server mode it can be provided per-request by the Chrome extension.
	if len(config.DestDir) == 0 {
		return configuration{}, errors.New("please provide a value for the 'dest_dir' field in the config file")
	}
	if len(config.PreKey) == 0 {
		return configuration{}, errors.New("please provide a value for the 'pre_key' field in the config file")
	}
	if len(config.Iv) == 0 {
		return configuration{}, errors.New("please provide a value for the 'iv' field in the config file")
	}
	return config, nil
}
