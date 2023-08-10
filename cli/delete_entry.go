package cli

import (
	"context"
	msg "github.com/ArthurHlt/messages"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
)

type DeleteEntry struct {
	FQDN string `short:"n" long:"fqdn" description:"FQDN of the entry" required:"true"`

	client gslbsvc.GSLBClient
}

func (c *DeleteEntry) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var deleteEntry DeleteEntry

func (c *DeleteEntry) Execute([]string) error {
	_, err := c.client.DeleteEntry(context.Background(), &gslbsvc.DeleteEntryRequest{
		Fqdn: c.FQDN,
	})
	if err != nil {
		return err
	}
	msg.Successf("Entry %s deleted", msg.Cyan(c.FQDN))
	return nil
}

func init() {
	desc := "Delete entry."
	cmd, err := parser.AddCommand(
		"delete-entry",
		desc,
		desc,
		&deleteEntry)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"de"}
}
