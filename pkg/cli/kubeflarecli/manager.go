package kubeflarecli

import (
	"os"

	"github.com/replicatedhq/kubeflare/pkg/apis"
	apitokencontroller "github.com/replicatedhq/kubeflare/pkg/controller/apitoken"
	ratelimitcontroller "github.com/replicatedhq/kubeflare/pkg/controller/ratelimit"
	wafrulecontroller "github.com/replicatedhq/kubeflare/pkg/controller/webapplicationfirewallrule"
	zonecontroller "github.com/replicatedhq/kubeflare/pkg/controller/zone"
	"github.com/replicatedhq/kubeflare/pkg/logger"
	"github.com/replicatedhq/kubeflare/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func ManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "manager",
		Short:         "runs the kubeflare manager (in cluster controller)",
		Long:          `...`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Infof("Starting kubeflare manager version %+v", version.GetBuild())

			v := viper.GetViper()

			if v.GetString("log-level") == "debug" {
				logger.Info("setting log level to debug")
				logger.SetDebug()
			}

			// Get a config to talk to the apiserver
			cfg, err := config.GetConfig()
			if err != nil {
				logger.Error(err)
				os.Exit(1)
			}

			// Create a new Cmd to provide shared dependencies and start components
			options := manager.Options{
				LeaderElection:   v.GetBool("leader-elect"),
				LeaderElectionID: "leaderelection.kubeflare.io",
			}

			mgr, err := manager.New(cfg, options)
			if err != nil {
				logger.Error(err)
				os.Exit(1)
			}

			// Setup Scheme for all resources
			if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
				logger.Error(err)
				os.Exit(1)
			}

			logger.Info("Starting Kubeflare Security Operator (WAF + Rate Limits)")

			// Enable API Token protection based on flag
			protectAPIToken := v.GetBool("protect-apitoken")
			// For now, we're disabling these controllers to get a working build
			// We'll re-enable them in Phase 3 and 4

			if false && protectAPIToken {
				logger.Info("API token protection enabled (disabled in MVP)")
				// err = apitokencontroller.Add(mgr)
				// if err != nil {
				//	logger.Error(err)
				//	os.Exit(1)
				// }
			}

			// Will re-enable Zone controller in Phase 4
			logger.Info("Zone controller disabled in MVP")
			// if err := zonecontroller.Add(mgr, protectAPIToken); err != nil {
			//	logger.Error(err)
			//	os.Exit(1)
			// }

			// Will re-enable WAF controller in Phase 3
			logger.Info("WAF controller disabled in MVP")
			// if err := wafrulecontroller.Add(mgr); err != nil {
			//	logger.Error(err)
			//	os.Exit(1)
			// }

			// Add Rate Limiting controller
			if err := ratelimitcontroller.Add(mgr); err != nil {
				logger.Error(err)
				os.Exit(1)
			}

			// Start the Cmd
			if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
				logger.Error(err)
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().String("metrics-addr", ":8088", "The address the metric endpoint binds to.")
	cmd.Flags().Bool("leader-elect", true, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")
	cmd.Flags().Bool("protect-apitoken", false, "Protect APIToken from deletion if it is referenced by other managed resources.")

	return cmd
}
