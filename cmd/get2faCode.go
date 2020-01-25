package cmd

import (
	"fmt"
	"log"

	"github.com/asishrs/smartthings-ringalarmv2/httputil"
	"github.com/spf13/cobra"
)

// get2faCodeCmd represents the get2faCode command
var get2faCodeCmd = &cobra.Command{
	Use:   "get2faCode",
	Short: "Get Two Factor Authentication Code (2FA) from Ring",
	Long: `Application requires a Two Factor Authentication Code (2FA) from Ring.
You need to enable the 2FA in Ring before you try this. 
Check https://support.ring.com/hc/en-us/articles/360024818291 if you need more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		user := cmd.Flag("user")
		password := cmd.Flag("password")
		makeAuthRequest(user.Value.String(), password.Value.String())
	},
}

func makeAuthRequest(user, password string) {
	response, err := httputil.AuthRequest("https://oauth.ring.com/oauth/token", httputil.OAuthRequest{"ring_official_ios", "password", password, "client", user}, "")
	log.Printf("OAuthResponse - %v", response)
	if err != nil {
		fmt.Println("Unable to authenticate. Please check your user name and password")
	} else if response.Error != "" {
		fmt.Printf("Unable to authenticate. \nRing API Error - %v : %v\n", response.Error, response.ErrorDescription)
	} else if response.Phone != "" {
		fmt.Println("You will be receiving a Text message on the registered Phone number.")
	}
}

func init() {
	rootCmd.AddCommand(get2faCodeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// get2faCodeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	get2faCodeCmd.Flags().StringP("user", "u", "", "Ring Account User Name (Email Address)")
	get2faCodeCmd.Flags().StringP("password", "p", "", "Ring Account Password")
}
