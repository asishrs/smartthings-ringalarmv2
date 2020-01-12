package cmd

import (
	"fmt"

	"github.com/asishrs/smartthings-ringalarmv2/httputil"
	"github.com/spf13/cobra"
)

// getRefreshKeyCmd represents the getRefreshKey command
var getRefreshKeyCmd = &cobra.Command{
	Use:   "getRefreshKey",
	Short: "Get Ring Alarm Refresh Key",
	Long: `With 2FA enabled Ring requires us to use a special refresh key that can be used 
long-term with 2FA turned on. A refresh key does not expire and allows us to 
bypass email/password/2FA altogether.`,
	Run: func(cmd *cobra.Command, args []string) {
		user := cmd.Flag("user")
		password := cmd.Flag("password")
		code := cmd.Flag("2facode")
		getRefreshToken(user.Value.String(), password.Value.String(), code.Value.String())
	},
}

func getRefreshToken(user string, password string, code string) {
	response, err := httputil.AuthRequest("https://oauth.ring.com/oauth/token", httputil.OAuthRequest{"ring_official_ios", "password", password, "client", user}, code)
	if err != nil {
		fmt.Println("Unable to authenticate. Please check your user name, password and 2FA code")
	} else if response.Error != "" {
		fmt.Printf("Unable to authenticate. \nRing API Error - %v\n", response.Error)
	} else if response.RefreshToken != "" {
		fmt.Printf("Your Refresh Token - \n%v\n\n", response.RefreshToken)
		fmt.Println("*** WARNING *** \n Refresh Token is equally powerful as your user-name/password as it can be used to access your account. DO NOT share this with anyone.\nNote Refresh Token, you will be asked to input this in the RingAlarm Smartthings App.")
	}
}

func init() {
	rootCmd.AddCommand(getRefreshKeyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getRefreshKeyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	getRefreshKeyCmd.Flags().StringP("user", "u", "", "Ring Account User Name (Email Address)")
	getRefreshKeyCmd.Flags().StringP("password", "p", "", "Ring Account Password")
	getRefreshKeyCmd.Flags().StringP("2facode", "c", "", "Ring Two Factor Authentication (2FA) Code (Received from Text message)")
}
