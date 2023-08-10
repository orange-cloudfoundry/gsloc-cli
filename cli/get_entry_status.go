package cli

import (
	"context"
	msg "github.com/ArthurHlt/messages"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
)

type GetEntryStatus struct {
	FQDN string `short:"n" long:"fqdn" description:"FQDN of the entry" required:"true"`
	Json bool   `short:"j" long:"json" description:"Format in json instead of human table readable."`

	client gslbsvc.GSLBClient
}

func (c *GetEntryStatus) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var getEntryStatus GetEntryStatus

func (c *GetEntryStatus) Execute([]string) error {
	msg.UseStderr()
	msg.Infof("Entry %s configuration", msg.Cyan(c.FQDN))
	msg.Printf("━━━━━\n")
	msg.UseStdout()
	entResp, err := c.client.GetEntryStatus(context.Background(), &gslbsvc.GetEntryStatusRequest{
		Fqdn: c.FQDN,
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
	desc := "Get entry with status."
	cmd, err := parser.AddCommand(
		"get-entry-status",
		desc,
		desc,
		&getEntryStatus)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"ges", "entry"}
}
