package cli

import (
	"context"
	"github.com/ArthurHlt/go-flags"
	"github.com/orange-cloudfoundry/gsloc-cli/app"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
	"os"
)

type MemberMap struct {
	Ip       string `mapstructure:"ip"`
	Ratio    int    `mapstructure:"ratio"`
	DC       string `mapstructure:"dc"`
	Disabled bool   `mapstructure:"disabled"`
}

type FQDN struct {
	content string
}

func (n *FQDN) String() string {
	return Fqdn(n.content)
}

func (n *FQDN) Complete(match string) []flags.Completion {
	opts.ConfigPath = defaultConfigPath
	if os.Getenv("GSLOC_CONFIG_PATH") != "" {
		opts.ConfigPath = os.Getenv("GSLOC_CONFIG_PATH")
	}

	clientConn, err := app.CreateConnFromFile(ExpandConfigPath())
	if err != nil {
		return []flags.Completion{
			{Item: "Error: " + err.Error()},
		}
	}
	defer func() {
		clientConn.Close()
	}()

	client := app.MakeClient(clientConn)
	resp, err := client.ListEntries(context.Background(), &gslbsvc.ListEntriesRequest{
		Prefix: match,
	})
	if err != nil {
		return []flags.Completion{
			{Item: "Error: " + err.Error()},
		}
	}
	completions := make([]flags.Completion, 0)
	for _, entry := range resp.GetEntries() {
		completions = append(completions, flags.Completion{
			Item: entry.GetEntry().GetFqdn(),
		})
	}
	return completions
}

func (n *FQDN) UnmarshalFlag(value string) error {
	n.content = value
	return nil
}
