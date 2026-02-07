package sentinel

import (
	"log/slog"

	"github.com/MatteoMori/sentinel/pkg/sentinel"
	sentinelShared "github.com/MatteoMori/sentinel/pkg/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var config sentinelShared.Config

var startSentinel = &cobra.Command{
	Use:     "start",
	Aliases: []string{"start"},
	Short:   "Start Sentinel controller",
	Args:    cobra.ExactArgs(0), // 0 arguments
	Run: func(cmd *cobra.Command, args []string) {
		sentinel.Start(config)
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().IntP("verbosity", "v", 0, "verbosity level (0-2)")

	// Viper bindings to Flags
	// This allows Viper to read the flags set by Cobra and use them in the configuration
	viper.BindPFlag("namespaceSelector", rootCmd.PersistentFlags().Lookup("namespaceSelector"))
	viper.BindPFlag("metricsPort", rootCmd.PersistentFlags().Lookup("metricsPort"))
	viper.BindPFlag("verbosity", rootCmd.PersistentFlags().Lookup("verbosity"))

	// Viper defaults
	viper.SetDefault("namespaceSelector", map[string]string{"sentinel.io/controlled": "enabled"})
	viper.SetDefault("metricsPort", "9090") // Default port for Prometheus metrics endpoint
	viper.SetDefault("verbosity", 0)
	viper.SetDefault("extraLabels", []sentinelShared.ExtraLabel{}) // Empty by default

	// Start the sentinel command
	rootCmd.AddCommand(startSentinel)
}

/*
	Sentinel Config

Here is where we initialize the Sentinel configuration.
This configuration is used to set up the controller, metrics, and other parameters.
- Ideally, all Sentinel parameters should have an equivalent here so that humans can override as they want.
*/
func initConfig() {
	viper.SetConfigName("sentinel") // Name of config file without extension
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/sentinel") // Config file expected location

	/*
		Viper will read environment variables and use them as configuration values if they match your config keys.
		EXAMPLE:
		export METRICSPORT=12345 --> Viper will use the value from the environment variable instead of the default or config file.
	*/
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		slog.Info("Config file loaded", "file", viper.ConfigFileUsed())
	} else {
		slog.Warn("No config file found, using environment variables and defaults", "err", err)
	}

	// Load into config struct
	if err := viper.Unmarshal(&config); err != nil {
		slog.Error("Unable to decode config into struct", "err", err)
	}

	// Apply fallback defaults if any field is missing
	//shared.ApplyDefaultConfig(&config)
}
