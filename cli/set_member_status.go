package cli

import (
	"context"
	"fmt"
	msg "github.com/ArthurHlt/messages"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
)

type SetMemberStatus struct {
	FQDN *FQDN  `short:"f" long:"fqdn" description:"FQDN of the entry fd."`
	Ip   string `short:"i" long:"ip" description:"IP of the member to disable."`

	DC     string   `short:"d" long:"dc" description:"Datacenter of the member to add."`
	Tags   []string `short:"t" long:"tag" description:"Filter by tag(s) (can be set multiple times)."`
	Prefix string   `short:"p" long:"prefix" description:"Filter by prefix/fqdn."`
	DryRun bool     `long:"dry-run" description:"Do not apply changes, just show what would be done."`
	State  string   `short:"s" long:"state" description:"State to set." choice:"enable" choice:"disable" required:"true"`
	Force  bool     `long:"force" description:"Force create entry without confirmation"`
	Json   bool     `short:"j" long:"json" description:"Format in json instead of human table readable."`
	client gslbsvc.GSLBClient
}

func (c *SetMemberStatus) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var setMemberStatus SetMemberStatus

func (c *SetMemberStatus) Execute([]string) error {
	state := gslbsvc.MemberState_ENABLED
	if c.State == "disable" {
		state = gslbsvc.MemberState_DISABLED
	}
	resp, err := c.client.SetMembersStatus(context.Background(), &gslbsvc.SetMembersStatusRequest{
		Prefix: c.Prefix,
		Ip:     c.Ip,
		Dc:     c.DC,
		Tags:   c.Tags,
		Status: state,
		DryRun: c.DryRun,
	})
	if err != nil {
		return err
	}
	if c.Json {
		return PrintProtoJson(resp)
	}
	stateText := msg.Green("Enabled").String()
	if state == gslbsvc.MemberState_DISABLED {
		stateText = msg.Red("Disabled").String()
	}
	if c.DryRun {
		msg.Info("This is a dry run, nothing has been done.")
		msg.Infof("This will %s those members:", stateText)
	} else {
		msg.Warning("This is not a dry run, changes has been applied.")
		msg.Warning("You may wait few seconds before changes are applied.")
		msg.Infof("Members below has been %s :", stateText)
	}
	table := MakeTableWriter([]string{"FQDN", "IPs"})
	table.SetAutoWrapText(false)
	for _, info := range resp.GetUpdated() {
		content := ""
		for _, ip := range info.GetIps() {
			content += fmt.Sprintf("%s\n", ip)
		}
		table.Append([]string{info.GetFqdn(), content})
	}
	table.Render()
	return nil
}

func init() {
	desc := "Disable or enable multiple or one member in one or multiple entries."
	cmd, err := parser.AddCommand(
		"set-member-status",
		desc,
		desc,
		&setMemberStatus)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"sms"}
}
