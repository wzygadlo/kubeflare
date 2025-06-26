package kubeflarecli

import (
	"os"

	"github.com/replicatedhq/kubeflare/pkg/apis"
	// accessapplicationcontroller "github.com/replicatedhq/kubeflare/pkg/controller/accessapplication"
	// apitokencontroller "github.com/replicatedhq/kubeflare/pkg/controller/apitoken"
	// dnsrecordcontroller "github.com/replicatedhq/kubeflare/pkg/controller/dnsrecord"
	// pagerulecontroller "github.com/replicatedhq/kubeflare/pkg/controller/pagerule"
	ratelimitcontroller "github.com/replicatedhq/kubeflare/pkg/controller/ratelimit"
	// wafrulecontroller "github.com/replicatedhq/kubeflare/pkg/controller/webapplicationfirewallrule"
	// workerroutecontroller "github.com/replicatedhq/kubeflare/pkg/controller/workerroute"
	// zonecontroller "github.com/replicatedhq/kubeflare/pkg/controller/zone"
	"github.com/replicatedhq/kubeflare/pkg/logger"
	"github.com/replicatedhq/kubeflare/pkg/version"
	// "github.com/replicatedhq/kubeflare/pkg/webhook"
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

			// Temporarily disable problematic controllers to focus on rate limiting
			logger.Info("Starting with rate limiting controller only")

			if err := ratelimitcontroller.Add(mgr); err != nil {
				logger.Error(err)
				os.Exit(1)
			}

			// TODO: Re-enable other controllers after fixing Cloudflare SDK compatibility
			// protectAPIToken := v.GetBool("protect-apitoken")
			// if protectAPIToken {
			// 	err = apitokencontroller.Add(mgr)
			// 	if err != nil {
			// 		logger.Error(err)
			// 		os.Exit(1)
			// 	}
			// }
			// if err := zonecontroller.Add(mgr, protectAPIToken); err != nil { ... }
			// if err := dnsrecordcontroller.Add(mgr); err != nil { ... }
			// if err := pagerulecontroller.Add(mgr); err != nil { ... }
			// if err := accessapplicationcontroller.Add(mgr); err != nil { ... }
			// if err := wafrulecontroller.Add(mgr); err != nil { ... }
			// if err := workerroutecontroller.Add(mgr); err != nil { ... }
			// if err := webhook.AddToManager(mgr); err != nil { ... }

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
