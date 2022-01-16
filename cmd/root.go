package cmd

import (
	"fmt"
	"github.com/markusressel/keyboard-backlight-daemon/internal/config"
	"github.com/markusressel/keyboard-backlight-daemon/internal/light"
	"github.com/markusressel/keyboard-backlight-daemon/internal/service"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
)

var (
	cfgFile string
	noColor bool
	noStyle bool
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "keyboard-backlight-daemon",
	Short: "A daemon to control the keyboard-backlight based on user activity.",
	// this is the default command to run when no subcommand is specified
	Run: func(cmd *cobra.Command, args []string) {
		setupUi()
		printHeader()

		config.ReadConfigFile()

		var lightPath string
		if config.CurrentConfig.BacklightPath != nil {
			lightPath = *config.CurrentConfig.BacklightPath
		} else {
			p := light.DetectKeyboardBacklight()
			if p == nil {
				panic("No keyboard backlight found")
			} else {
				lightPath = *p
				fmt.Printf("Detected keyboard backlight: %s\n", lightPath)
			}
		}
		l := light.NewLight(lightPath)

		s := service.NewKbdService(config.CurrentConfig, l)

		s.Run()
	},
}

func setupUi() {
	if noColor {
		pterm.DisableColor()
	}
	if noStyle {
		pterm.DisableStyling()
	}
}

// Print a large text with the LetterStyle from the standard theme.
func printHeader() {
	err := pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("kbd", pterm.NewStyle(pterm.FgWhite)),
	).Render()
	if err != nil {
		fmt.Println("kbd")
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.OnInitialize(func() {
		config.InitConfig(cfgFile)
	})

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/kbd/kbd.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&noColor, "no-color", "", false, "Disable all terminal output coloration")
	rootCmd.PersistentFlags().BoolVarP(&noStyle, "no-style", "", false, "Disable all terminal output styling")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "More verbose output")

	if err := rootCmd.Execute(); err != nil {

		fmt.Printf("Error Executing daemon: %v", err)
		os.Exit(1)
	}
}
