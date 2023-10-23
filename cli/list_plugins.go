package cli

import (
	"context"
	msg "github.com/ArthurHlt/messages"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
	"google.golang.org/protobuf/types/known/emptypb"
	"sort"
)

type ListPlugins struct {
	Json bool `short:"j" long:"json" description:"Format in json instead of human table readable."`

	client gslbsvc.GSLBClient
}

func (c *ListPlugins) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

var listPlugins ListPlugins

func (c *ListPlugins) Execute([]string) error {
	plugResp, err := c.client.ListPluginHealthChecks(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}

	if c.Json {
		return PrintProtoJson(plugResp)
	}

	if len(plugResp.GetPluginHealthChecks()) == 0 {
		msg.Info("No plugins found.")
		return nil
	}

	plugins := plugResp.GetPluginHealthChecks()
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].GetName() < plugins[j].GetName()
	})

	table := MakeTableWriter([]string{"Name", "Description"})
	table.SetAutoWrapText(false)
	for _, plugin := range plugins {
		table.Append([]string{plugin.GetName(), plugin.GetDescription()})
	}
	table.Render()
	return nil
}

func init() {
	desc := "List healthchecks plugins available on server."
	cmd, err := parser.AddCommand(
		"healthchecks-plugins",
		desc,
		desc,
		&listPlugins)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"hps"}
}
