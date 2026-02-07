package sentinel

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sentinel",
	Short: "Sentinel is a Kubernetes controller that tracks container images across workloads",
	Long:  `Sentinel is a Kubernetes controller that tracks container images across your cluster and exposes them as Prometheus metrics`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}
