package cli

import (
	"context"
	msg "github.com/ArthurHlt/messages"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
)

type GetEntry struct {
	FQDN *FQDN `positional-args:"true" positional-arg-name:"'fqdn'" required:"true"`
	Json bool  `short:"j" long:"json" description:"Format in json instead of human table readable."`

	client gslbsvc.GSLBClient
}

func (c *GetEntry) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var getEntry GetEntry

func (c *GetEntry) Execute([]string) error {
	msg.UseStderr()
	msg.Infof("Entry %s configuration", msg.Cyan(c.FQDN))
	msg.Printf("━━━━━\n")
	msg.UseStdout()
	entResp, err := c.client.GetEntry(context.Background(), &gslbsvc.GetEntryRequest{
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
	desc := "Get entry."
	cmd, err := parser.AddCommand(
		"entry",
		desc,
		desc,
		&getEntry)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"e"}
}
