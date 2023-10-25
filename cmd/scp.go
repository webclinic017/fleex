package cmd

import (
	"path/filepath"
	"strings"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

// scpCmd represents the scp command
var scpCmd = &cobra.Command{
	Use:   "scp",
	Short: "Send a file/folder to a fleet using SCP",
	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		usernameFlag, _ := cmd.Flags().GetString("username")
		sourceFlag, _ := cmd.Flags().GetString("source")
		portFlag, _ := cmd.Flags().GetInt("port")
		destinationFlag, _ := cmd.Flags().GetString("destination")
		nameFlag, _ := cmd.Flags().GetString("name")

		home, _ := homedir.Dir()

		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}
		providerFlag = globalConfig.Settings.Provider

		provider := controller.GetProvider(providerFlag)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		providerInfo := globalConfig.Providers[providerFlag]
		if portFlag != -1 {
			providerInfo.Port = portFlag
		}
		if usernameFlag != "" {
			providerInfo.Username = usernameFlag
		}

		if strings.HasPrefix(destinationFlag, home) {
			if home != "/root" {
				destinationFlag = filepath.Join("/home", usernameFlag, strings.TrimPrefix(destinationFlag, home))
			}
		}

		newController := controller.NewController(globalConfig)

		fleets := newController.GetFleet(nameFlag)
		if len(fleets) == 0 {
			utils.Log.Fatal("Box not found")
		}
		for _, box := range fleets {
			if box.Label == nameFlag {
				controller.SendSCP(sourceFlag, usernameFlag, destinationFlag, box.IP, portFlag, globalConfig.SSHKeys.PrivateFile)
				return
			}
		}

		for _, box := range fleets {
			if strings.HasPrefix(box.Label, nameFlag) {
				controller.SendSCP(sourceFlag, usernameFlag, destinationFlag, box.IP, portFlag, globalConfig.SSHKeys.PrivateFile)
			}
		}

		utils.Log.Info("SCP completed, you can find your files in " + destinationFlag)
	},
}

func init() {
	rootCmd.AddCommand(scpCmd)

	scpCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")
	scpCmd.Flags().StringP("name", "n", "pwn", "Fleet name")
	scpCmd.Flags().StringP("username", "U", "", "Username")
	scpCmd.Flags().IntP("port", "", -1, "SSH port")
	scpCmd.Flags().StringP("source", "s", "", "Source file / folder")
	scpCmd.Flags().StringP("destination", "d", "", "Destination file / folder")

	scpCmd.MarkFlagRequired("source")
	scpCmd.MarkFlagRequired("destination")

}
