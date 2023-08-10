package cli

import (
	"context"
	"fmt"
	msg "github.com/ArthurHlt/messages"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
	"github.com/sourcegraph/conc/pool"
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
	entsResp, err := c.client.ListEntries(context.Background(), &gslbsvc.ListEntriesRequest{
		Tags:   c.Tags,
		Prefix: c.Prefix,
	})
	if err != nil {
		return err
	}
	var entsStatus []*gslbsvc.GetEntryStatusResponse
	var chanEntsStatus = make(chan *gslbsvc.GetEntryStatusResponse, 100)
	p := pool.New().WithMaxGoroutines(10)
	done := make(chan struct{})
	go func() {
		for entStatus := range chanEntsStatus {
			entsStatus = append(entsStatus, entStatus)
		}
		done <- struct{}{}
	}()
	for _, ent := range entsResp.GetEntries() {
		ent := ent
		p.Go(func() {
			entStatus, err := c.client.GetEntryStatus(context.Background(), &gslbsvc.GetEntryStatusRequest{
				Fqdn: ent.GetEntry().GetFqdn(),
			})
			if err != nil {
				msg.Fatal(err.Error())
			}
			chanEntsStatus <- entStatus
		})
	}
	p.Wait()
	close(chanEntsStatus)
	<-done
	if c.Json {
		return PrintProtoListJson[*gslbsvc.GetEntryStatusResponse](entsStatus)
	}

	if len(entsStatus) == 0 {
		msg.Info("No entries found.")
		return nil
	}

	dcResp, err := c.client.ListDcs(context.Background(), &gslbsvc.ListDcsRequest{})
	if err != nil {
		return err
	}

	table := MakeTableWriter(append([]string{"FQDN"}, dcResp.GetDcs()...))
	table.SetAutoWrapText(false)
	for _, entStatus := range entsStatus {
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
		}
		if entMemberStatus.GetStatus() == gslbsvc.MemberStatus_OFFLINE {
			dcContent += msg.Red(entMemberStatus.GetIp()).String() + " (Offline)\n"
		}
		if entMemberStatus.GetStatus() == gslbsvc.MemberStatus_CHECK_FAILED {
			dcContent += msg.Red(entMemberStatus.GetIp()).String() +
				fmt.Sprintf(" (Check failed: %s)\n", entMemberStatus.GetFailureReason())
		}
	}
	return dcContent
}

func init() {
	desc := "List entries with status."
	cmd, err := parser.AddCommand(
		"list-entries-status",
		desc,
		desc,
		&listEntriesStatus)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"les", "list"}
}
