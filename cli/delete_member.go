package cli

import (
	"context"
	msg "github.com/ArthurHlt/messages"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
)

type DeleteMember struct {
	FQDN   *FQDN  `positional-args:"true" positional-arg-name:"'fqdn'" required:"true"`
	Ip     string `short:"i" long:"ip" description:"IP of the member to delete." required:"true"`
	client gslbsvc.GSLBClient
}

func (c *DeleteMember) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var deleteMember DeleteMember

func (c *DeleteMember) Execute([]string) error {
	_, err := c.client.DeleteMember(context.Background(), &gslbsvc.DeleteMemberRequest{
		Fqdn: c.FQDN.String(),
		Ip:   c.Ip,
	})
	if err != nil {
		return err
	}
	msg.Successf("Member %s deleted", msg.Cyan(c.FQDN))
	return nil
}

func init() {
	desc := "Delete member."
	cmd, err := parser.AddCommand(
		"delete-member",
		desc,
		desc,
		&deleteMember)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"dm"}
}
