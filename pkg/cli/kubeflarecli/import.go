package kubeflarecli

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/pkg/errors"
	kubeflarescheme "github.com/replicatedhq/kubeflare/pkg/client/kubeflareclientset/scheme"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func ImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "import",
		Short:         "import existing security rules from cloudflare into custom resources",
		Long:          `Import existing Web Application Firewall (WAF) rules and Rate Limiting rules from Cloudflare into Kubernetes custom resources`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			_, err := os.Stat(v.GetString("output-dir"))
			if os.IsNotExist(err) {
				if err := os.MkdirAll(v.GetString("output-dir"), 0755); err != nil {
					return errors.Wrap(err, "mkdir")
				}
			} else if err != nil {
				return errors.Wrap(err, "stat")
			}

			cf, err := cloudflare.NewWithAPIToken(v.GetString("api-token"))
			if err != nil {
				return errors.Wrap(err, "create clouflare client")
			}

			zoneID, err := cf.ZoneIDByName(v.GetString("zone"))
			if err != nil {
				return errors.Wrap(err, "get zone id")
			}

			kubeflarescheme.AddToScheme(scheme.Scheme)
			s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

			// TODO: Implement WAF rule import using Rulesets API

			// TODO: Implement Rate Limit rule import using Rulesets API

			return fmt.Errorf("Import functionality for WAF rules and Rate Limits not yet implemented with the latest Cloudflare SDK v4.5.1")

			// NOTE: This return is unreachable, previous return will exit the function
			// Future implementation will focus on WAF rules and Rate Limits only
		},
	}

	cmd.Flags().String("api-token", "", "cloudflare api token")
	cmd.MarkFlagRequired("api-token")

	cmd.Flags().String("zone", "", "dns zone to import")
	cmd.MarkFlagRequired("zone")

	cmd.Flags().String("output-dir", filepath.Join(".", "imported"), "output dir to write files to")

	cmd.Flags().Bool("waf-rules", true, "when set, import existing web application firewall rules from the zone")
	cmd.Flags().Bool("rate-limits", true, "when set, import existing rate limiting rules from the zone")

	return cmd
}
