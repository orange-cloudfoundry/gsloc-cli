package cli

import (
	"context"
	msg "github.com/ArthurHlt/messages"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
)

type GetMember struct {
	FQDN *FQDN  `positional-args:"true" positional-arg-name:"'fqdn'" required:"true"`
	Json bool   `short:"j" long:"json" description:"Format in json instead of human table readable."`
	Ip   string `short:"i" long:"ip" description:"IP of the member to get." required:"true"`

	client gslbsvc.GSLBClient
}

func (c *GetMember) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var getMember GetMember

func (c *GetMember) Execute([]string) error {
	msg.UseStderr()
	msg.Infof("Member %s configuration", msg.Cyan(c.FQDN))
	msg.Printf("━━━━━\n")
	msg.UseStdout()
	entResp, err := c.client.GetMember(context.Background(), &gslbsvc.GetMemberRequest{
		Fqdn: c.FQDN.String(),
		Ip:   c.Ip,
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
	desc := "Get member."
	cmd, err := parser.AddCommand(
		"member",
		desc,
		desc,
		&getMember)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"m"}
}
