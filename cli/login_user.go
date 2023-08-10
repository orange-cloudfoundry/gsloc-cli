package cli

import (
	"github.com/AlecAivazis/survey/v2"
	msg "github.com/ArthurHlt/messages"
	"github.com/orange-cloudfoundry/gsloc-cli/app"
	"strings"
)

type LoginUser struct {
	Host              string `short:"t" long:"host" description:"Host of gsloc" env:"GSLOC_HOST"`
	Username          string `short:"u" long:"username" description:"Username" env:"GSLOC_USERNAME"`
	Password          string `short:"p" long:"password" description:"Password" env:"GSLOC_PASSWORD"`
	SkipSslValidation bool   `short:"k" long:"skip-ssl-validation" description:"Skip SSL validation"`
}

var loginUser LoginUser

func (c *LoginUser) Execute([]string) error {
	hostSplit := strings.Split(c.Host, ":")
	if c.Host != "" && len(hostSplit) == 1 {
		c.Host = c.Host + ":443"
	}
	currentUsername := app.GetCurrentUsername(ExpandConfigPath())
	var qs []*survey.Question
	if c.Username == "" {
		qs = append(qs, &survey.Question{
			Name: "username",
			Prompt: &survey.Input{
				Message: "Username:",
				Default: currentUsername,
			},
		})
	}
	if c.Password == "" {
		qs = append(qs, &survey.Question{
			Name: "password",
			Prompt: &survey.Password{
				Message: "Password:",
			},
		})
	}
	answers := struct {
		Username string
		Password string
	}{}
	if len(qs) > 0 {
		// perform the questions
		err := survey.Ask(qs, &answers)
		if err != nil {
			return err
		}
		if answers.Username != "" {
			c.Username = answers.Username
		}
		if answers.Password != "" {
			c.Password = answers.Password
		}
	}
	_, err := app.CreateConn(ExpandConfigPath(), c.Host, c.Username, c.Password, c.SkipSslValidation)
	if err != nil {
		return err
	}
	msg.Success("Login successful.")
	return nil

}

func init() {
	desc := "Login"
	cmd, err := parser.AddCommand(
		"login",
		desc,
		desc,
		&loginUser)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"login"}
}
