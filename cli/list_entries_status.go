package cli

import (
	"context"
	"fmt"
	msg "github.com/ArthurHlt/messages"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
	"strings"
)

type ListEntriesStatus struct {
	Json bool `short:"j" long:"json" description:"Format in json instead of human table readable."`

	Tags   []string `short:"t" long:"tag" description:"Filter by tag(s) (can be set multiple times)."`
	Prefix string   `short:"p" long:"prefix" description:"Filter by prefix."`

	client gslbsvc.GSLBClient
}

func (c *ListEntriesStatus) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var listEntriesStatus ListEntriesStatus

func (c *ListEntriesStatus) Execute([]string) error {
	entsResp, err := c.client.ListEntriesStatus(context.Background(), &gslbsvc.ListEntriesStatusRequest{
		Tags:   c.Tags,
		Prefix: c.Prefix,
	})
	if err != nil {
		return err
	}

	if c.Json {
		return PrintProtoListJson[*gslbsvc.GetEntryStatusResponse](entsResp.GetEntriesStatus())
	}

	if len(entsResp.GetEntriesStatus()) == 0 {
		msg.Info("No entries found.")
		return nil
	}

	dcResp, err := c.client.ListDcs(context.Background(), &gslbsvc.ListDcsRequest{})
	if err != nil {
		return err
	}

	table := MakeTableWriter(append([]string{"FQDN"}, dcResp.GetDcs()...))
	table.SetAutoWrapText(false)
	for _, entStatus := range entsResp.GetEntriesStatus() {
		line := []string{entStatus.GetFqdn()}
		for _, dc := range dcResp.GetDcs() {
			dcContent := c.makeDcContent(entStatus.GetMembersIpv4(), dc)
			dcContent += "\n" + c.makeDcContent(entStatus.GetMembersIpv6(), dc)
			line = append(line, dcContent)
		}
		table.Append(line)
	}
	table.Render()
	return nil
}

func (c *ListEntriesStatus) makeDcContent(entMemberStatus []*gslbsvc.MemberStatus, dc string) string {
	dcContent := ""
	for _, entMemberStatus := range entMemberStatus {
		if entMemberStatus.GetDc() != dc {
			continue
		}
		if entMemberStatus.GetStatus() == gslbsvc.MemberStatus_ONLINE {
			dcContent += msg.Green(entMemberStatus.GetIp()).String() + " (Online)\n"
			continue
		}
		if entMemberStatus.GetStatus() == gslbsvc.MemberStatus_OFFLINE {
			dcContent += msg.Red(entMemberStatus.GetIp()).String() + " (Offline)\n"
			continue
		}
		reason := entMemberStatus.GetFailureReason()
		if entMemberStatus.GetStatus() == gslbsvc.MemberStatus_CHECK_FAILED && strings.Contains(reason, "disabled entry") {
			dcContent += msg.Yellow(entMemberStatus.GetIp()).String() + " (Disabled by User)\n"
			continue
		}
		dcContent += msg.Red(entMemberStatus.GetIp()).String() +
			fmt.Sprintf(" (Check failed: %s)\n", entMemberStatus.GetFailureReason())
	}
	return dcContent
}

func init() {
	desc := "List entries with status."
	cmd, err := parser.AddCommand(
		"entries-status",
		desc,
		desc,
		&listEntriesStatus)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"ess"}
}
