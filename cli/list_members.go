package cli

import (
	"context"
	"fmt"
	msg "github.com/ArthurHlt/messages"
	"github.com/olekukonko/tablewriter"
	"github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/api/config/entries/v1"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
)

type ListMembers struct {
	FQDN *FQDN `positional-args:"true" positional-arg-name:"'fqdn'" required:"true"`
	Json bool  `short:"j" long:"json" description:"Format in json instead of human table readable."`

	client gslbsvc.GSLBClient
}

func (c *ListMembers) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var listMembers ListMembers

func (c *ListMembers) Execute([]string) error {
	msg.UseStderr()
	msg.Infof("Member %s configuration", msg.Cyan(c.FQDN))
	msg.Printf("━━━━━\n")
	msg.UseStdout()
	entResp, err := c.client.ListMembers(context.Background(), &gslbsvc.ListMembersRequest{
		Fqdn: c.FQDN.String(),
	})
	if err != nil {
		return err
	}
	if c.Json {
		return PrintProtoJson(entResp)
	}
	table := MakeTableWriter([]string{"DC", "IP", "Ratio", "State"})
	table.SetAutoWrapText(false)
	c.addToTable(table, entResp.GetMembersIpv4())
	c.addToTable(table, entResp.GetMembersIpv6())
	table.Render()
	return nil
}

func (c *ListMembers) addToTable(table *tablewriter.Table, members []*entries.Member) {
	for _, member := range members {
		enabled := msg.Green("Enabled").String()
		if member.GetDisabled() {
			enabled = msg.Red("Disabled").String()
		}
		table.Append([]string{
			member.GetDc(),
			member.GetIp(),
			fmt.Sprintf("%d", member.GetRatio()),
			enabled,
		})
	}
}

func init() {
	desc := "List members."
	cmd, err := parser.AddCommand(
		"members",
		desc,
		desc,
		&listMembers)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"ms"}
}
