package cli

import (
	"context"
	msg "github.com/ArthurHlt/messages"
	"github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/api/config/entries/v1"
	hcconf "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/api/config/healthchecks/v1"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
)

type ListEntries struct {
	Json bool `short:"j" long:"json" description:"Format in json instead of human table readable."`

	Tags   []string `short:"t" long:"tag" description:"Filter by tag(s) (can be set multiple times)."`
	Prefix string   `short:"p" long:"prefix" description:"Filter by prefix."`

	client gslbsvc.GSLBClient
}

func (c *ListEntries) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var listEntries ListEntries

func (c *ListEntries) Execute([]string) error {
	entsResp, err := c.client.ListEntries(context.Background(), &gslbsvc.ListEntriesRequest{
		Tags:   c.Tags,
		Prefix: c.Prefix,
	})
	if err != nil {
		return err
	}

	if len(entsResp.Entries) == 0 {
		msg.Info("No entries found.")
		return nil
	}

	dcResp, err := c.client.ListDcs(context.Background(), &gslbsvc.ListDcsRequest{})
	if err != nil {
		return err
	}

	table := MakeTableWriter(append([]string{"FQDN", "Healthcheck"}, dcResp.GetDcs()...))
	table.SetAutoWrapText(false)
	for _, ent := range entsResp.GetEntries() {
		line := []string{ent.GetEntry().GetFqdn()}
		switch ent.GetHealthcheck().HealthChecker.(type) {
		case *hcconf.HealthCheck_HttpHealthCheck:
			line = append(line, "HTTP")
		case *hcconf.HealthCheck_TcpHealthCheck:
			line = append(line, "TCP")
		case *hcconf.HealthCheck_GrpcHealthCheck:
			line = append(line, "GRPC")
		default:
			line = append(line, "NONE")
		}

		for _, dc := range dcResp.GetDcs() {
			dcContent := c.makeDcContent(ent.GetEntry().GetMembersIpv4(), dc)
			dcContent += "\n" + c.makeDcContent(ent.GetEntry().GetMembersIpv6(), dc)
			line = append(line, dcContent)
		}
		table.Append(line)
	}
	table.Render()
	return nil
}

func (c *ListEntries) makeDcContent(members []*entries.Member, dc string) string {
	dcContent := ""
	for _, member := range members {
		if member.GetDc() != dc {
			continue
		}
		dcContent += member.GetIp() + "\n"
	}
	return dcContent
}

func init() {
	desc := "List entries."
	cmd, err := parser.AddCommand(
		"list-entries",
		desc,
		desc,
		&listEntries)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"le", "get-entries"}
}
