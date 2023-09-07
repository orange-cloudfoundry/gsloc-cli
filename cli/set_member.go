package cli

import (
	"context"
	msg "github.com/ArthurHlt/messages"
	"github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/api/config/entries/v1"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
	"google.golang.org/protobuf/proto"
	"strings"
)

type SetMember struct {
	FQDN     *FQDN  `positional-args:"true" positional-arg-name:"'fqdn'" required:"true"`
	Ip       string `short:"i" long:"ip" description:"IP of the member to add." required:"true"`
	DC       string `short:"d" long:"dc" description:"Datacenter of the member to add." required:"true"`
	Ratio    uint32 `short:"r" long:"ratio" description:"Ratio of the member to add."`
	Disabled bool   `short:"D" long:"disabled" description:"Disable the member to add."`

	Force bool `long:"force" description:"Force create entry without confirmation"`

	client gslbsvc.GSLBClient
}

func (c *SetMember) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var setMember SetMember

func (c *SetMember) Execute([]string) error {
	setMemberReq := &gslbsvc.SetMemberRequest{
		Fqdn: c.FQDN.String(),
		Member: &entries.Member{
			Dc:       c.DC,
			Ip:       c.Ip,
			Ratio:    c.Ratio,
			Disabled: c.Disabled,
		},
	}
	var previousEntry *gslbsvc.SetMemberRequest
	resp, err := c.client.GetMember(context.Background(), &gslbsvc.GetMemberRequest{
		Fqdn: c.FQDN.String(),
		Ip:   c.Ip,
	})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	if err == nil {
		previousEntry = &gslbsvc.SetMemberRequest{
			Fqdn:   c.FQDN.String(),
			Member: resp.GetMember(),
		}
		proto.Merge(setMemberReq, previousEntry)
	}
	confirm, err := DiffAndConfirm(previousEntry, setMemberReq, c.Force)
	if err != nil {
		return err
	}
	if !confirm {
		return nil
	}
	_, err = c.client.SetMember(context.Background(), setMemberReq)
	if err != nil {
		return err
	}
	msg.Successf("Member %s set successfully.", msg.Cyan(c.Ip))
	return nil
}

func init() {
	desc := "Create or update member on entry."
	cmd, err := parser.AddCommand(
		"set-member",
		desc,
		desc,
		&setMember)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"sm"}
}
