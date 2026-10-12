package config

import (
	"errors"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"io/fs"
	"path"
	"time"
)

type Configuration struct {
	BacklightPath     *string
	InputEventDevices []string
	IdleTimeout       time.Duration
}

var CurrentConfig Configuration

func InitConfig(cfgFile string) error {
	viper.SetConfigName("keyboard-backlight-daemon")

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			return fmt.Errorf("couldn't detect home directory: %w", err)
		}

		viper.AddConfigPath(".")
		viper.AddConfigPath(path.Join(home, ".config"))
		viper.AddConfigPath("/etc/keyboard-backlight-daemon/")
	}

	viper.AutomaticEnv() // read in environment variables that match

	setDefaultValues()
	return nil
}

func setDefaultValues() {
	viper.SetDefault("idleTimeout", 6*time.Second)
}

// ReadConfigFile reads the configuration file (if there is one) into CurrentConfig.
// A missing configuration file is not an error, defaults are used in that case.
func ReadConfigFile() error {
	if err := viper.ReadInConfig(); err == nil {
		// this is only populated _after_ ReadInConfig()
		fmt.Printf("Using configuration file at: %s\n", viper.ConfigFileUsed())
	} else {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	if err := LoadConfig(); err != nil {
		return err
	}
	return validateConfig(CurrentConfig)
}

// LoadConfig loads the configuration, including defaults, into CurrentConfig.
func LoadConfig() error {
	if err := viper.Unmarshal(&CurrentConfig); err != nil {
		return fmt.Errorf("unable to decode configuration: %w", err)
	}
	return nil
}

func validateConfig(c Configuration) error {
	if c.IdleTimeout <= 0 {
		return fmt.Errorf("invalid idleTimeout: %s (must be greater than zero)", c.IdleTimeout)
	}
	return nil
}
