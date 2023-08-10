package cli

import (
	"context"
	msg "github.com/ArthurHlt/messages"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
)

type ListDcs struct {
	Json bool `short:"j" long:"json" description:"Format in json instead of human table readable."`

	Tags   []string `short:"t" long:"tag" description:"Filter by tag(s) (can be set multiple times)."`
	Prefix string   `short:"p" long:"prefix" description:"Filter by prefix."`

	client gslbsvc.GSLBClient
}

func (c *ListDcs) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var listDcs ListDcs

func (c *ListDcs) Execute([]string) error {
	dcsResp, err := c.client.ListDcs(context.Background(), &gslbsvc.ListDcsRequest{})
	if err != nil {
		return err
	}

	if c.Json {
		return PrintProtoJson[*gslbsvc.ListDcsResponse](dcsResp)
	}

	if len(dcsResp.GetDcs()) == 0 {
		msg.Info("No datacenters found.")
		return nil
	}

	table := MakeTableWriter([]string{"DATACENTER"})
	table.SetAutoWrapText(false)
	for _, dc := range dcsResp.GetDcs() {
		table.Append([]string{dc})
	}
	table.Render()
	return nil
}

func init() {
	desc := "List datacenters."
	cmd, err := parser.AddCommand(
		"list-dcs",
		desc,
		desc,
		&listDcs)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"ld", "list-dc"}
}
