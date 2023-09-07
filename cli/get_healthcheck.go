package cli

import (
	"context"
	msg "github.com/ArthurHlt/messages"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
)

type GetHealthcheck struct {
	FQDN *FQDN `positional-args:"true" positional-arg-name:"'fqdn'" required:"true"`
	Json bool  `short:"j" long:"json" description:"Format in json instead of human table readable."`

	client gslbsvc.GSLBClient
}

func (c *GetHealthcheck) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var getHealthcheck GetHealthcheck

func (c *GetHealthcheck) Execute([]string) error {
	msg.UseStderr()
	msg.Infof("Healthcheck %s configuration", msg.Cyan(c.FQDN))
	msg.Printf("━━━━━\n")
	msg.UseStdout()
	entResp, err := c.client.GetHealthCheck(context.Background(), &gslbsvc.GetHealthCheckRequest{
		Fqdn: c.FQDN.String(),
	})
	if err != nil {
		return err
	}
	if c.Json {
		return PrintProtoJson(entResp)
	}

	return PrintProtoHuman(entResp)
}

func init() {
	desc := "Get healthcheck."
	cmd, err := parser.AddCommand(
		"healthcheck",
		desc,
		desc,
		&getHealthcheck)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"hc"}
}
