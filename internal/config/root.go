package config

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"os"
	"path"
	"time"
)

type Configuration struct {
	BacklightPath     *string
	InputEventDevices []string
	IdleTimeout       time.Duration
}

var CurrentConfig Configuration

func InitConfig(cfgFile string) {
	viper.SetConfigName("keyboard-backlight-daemon")

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Printf("Couldn't detect home directory: %v", err)
			os.Exit(1)
		}

		viper.AddConfigPath(".")
		viper.AddConfigPath(path.Join(home, ".config"))
		viper.AddConfigPath("/etc/keyboard-backlight-daemon/")
	}

	viper.AutomaticEnv() // read in environment variables that match

	setDefaultValues()
}

func setDefaultValues() {
	viper.SetDefault("idleTimeout", 6*time.Second)
}

func ReadConfigFile() {
	if err := viper.ReadInConfig(); err == nil {
		// this is only populated _after_ ReadInConfig()
		fmt.Printf("Using configuration file at: %s\n", viper.ConfigFileUsed())
		LoadConfig()
	}

	validateConfig()
}

func LoadConfig() {
	// load default configuration values
	err := viper.Unmarshal(&CurrentConfig)
	if err != nil {
		fmt.Printf("unable to decode into struct, %v", err)
		os.Exit(1)
	}
}

func validateConfig() {
	_ = &CurrentConfig
	// TODO:
}
