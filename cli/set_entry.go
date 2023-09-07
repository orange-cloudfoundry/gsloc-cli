package cli

import (
	"context"
	"fmt"
	"github.com/ArthurHlt/go-flags"
	msg "github.com/ArthurHlt/messages"
	"github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/api/config/core/v1"
	"github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/api/config/entries/v1"
	hcconf "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/api/config/healthchecks/v1"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
	gsloctype "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/type/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"strconv"
	"strings"
	"time"
)

type SetEntry struct {
	File flags.Filename `short:"f" long:"file" description:"Path to a json or yml file definition of entry" required:"true" default:"entry.yml"`

	FQDN              *FQDN               `positional-args:"true" positional-arg-name:"'fqdn'" required:"true"`
	LBAlgoPreferred   string              `short:"p" long:"lb-algo-preferred" description:"LB algo preferred" choice:"ROUND_ROBIN" choice:"TOPOLOGY" choice:"RATIO" choice:"RANDOM" default:"ROUND_ROBIN"`
	LBAlgoAlternate   string              `short:"a" long:"lb-algo-alternate" description:"LB algo alternate" choice:"ROUND_ROBIN" choice:"TOPOLOGY" choice:"RATIO" choice:"RANDOM" default:"ROUND_ROBIN"`
	LBAlgoFallback    string              `short:"b" long:"lb-algo-fallback" description:"LB algo fallback" choice:"ROUND_ROBIN" choice:"TOPOLOGY" choice:"RATIO" choice:"RANDOM" default:"ROUND_ROBIN"`
	MaxAnswerReturned uint32              `short:"m" long:"max-answer-returned" description:"Max answer returned" default:"0"`
	MembersIPv4       []map[string]string `short:"4" long:"members-ipv4" description:"Members ipv4 (can be set multiple time)"`
	MembersIPv6       []map[string]string `short:"6" long:"members-ipv6" description:"Members ipv6 (can be set multiple time)"`
	TTL               uint32              `long:"ttl" description:"TTL" default:"30"`
	Tags              []string            `short:"T" long:"tag" description:"Tag (can be set multiple time)"`

	HcTimeout   string `short:"o" long:"hc-timeout" description:"Healthcheck timeout" default:"10s"`
	HcInterval  string `short:"i" long:"hc-interval" description:"Healthcheck interval" default:"30s"`
	HcPort      uint32 `short:"P" long:"hc-port" description:"Healthcheck port" default:"80"`
	HcType      string `short:"t" long:"hc-type" description:"Healthcheck type" choice:"HTTP" choice:"TCP" choice:"GRPC" choice:"NO_HEALTHCHECK" default:"NO_HEALTHCHECK"`
	HcEnableTls bool   `long:"hc-enable-tls" description:"Enable tls during healthcheck"`

	HttpHost            string            `short:"H" long:"http-host" description:"HTTP healthcheck host"`
	HttpPath            string            `long:"http-path" description:"HTTP healthcheck path"`
	HttpCode            int               `short:"C" long:"http-code" description:"HTTP healthcheck code"`
	HttpCodeRange       string            `long:"http-code-range" description:"HTTP healthcheck range code (e.g.: 200-299)"`
	HttpSendPayload     string            `short:"S" long:"http-send-payload" description:"HTTP healthcheck send payload"`
	HttpReceivePayload  string            `short:"R" long:"http-receive-payload" description:"HTTP healthcheck receive payload"`
	HttpHeaders         map[string]string `short:"d" long:"http-headers" description:"HTTP healthcheck header (e.g.: 'X-Header:foo,X-Header2:bar')"`
	HttpMethod          string            `short:"M" long:"http-method" description:"HTTP healthcheck method" choice:"GET" choice:"HEAD" choice:"POST" choice:"PUT" choice:"DELETE" choice:"OPTIONS" choice:"TRACE" choice:"PATCH" default:"GET"`
	HttpCodecClientType string            `short:"c" long:"http-codec-client-type" description:"HTTP healthcheck codec client type" choice:"HTTP1" choice:"HTTP2" choice:"AUTO" default:"AUTO"`

	TcpSendPayload     string   `long:"tcp-send-payload" description:"TCP healthcheck send payload"`
	TcpReceivePayloads []string `long:"tcp-receive-payloads" description:"TCP healthcheck receive payloads (can be set multiple time)"`

	GRPCServiceName string `long:"grpc-service-name" description:"gRPC healthcheck service name"`
	GRPCAuthority   string `long:"grpc-authority" description:"gRPC healthcheck authority"`

	Force bool `long:"force" description:"Force create entry without confirmation"`

	client gslbsvc.GSLBClient
}

var setEntry SetEntry

func (c *SetEntry) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

func (c *SetEntry) Execute([]string) error {
	entryToSet, loaded, err := FileToProto[*gslbsvc.SetEntryRequest](string(c.File))
	if err != nil {
		return err
	}
	if entryToSet.Entry == nil {
		entryToSet.Entry = &entries.Entry{}
	}
	if entryToSet.Healthcheck == nil {
		entryToSet.Healthcheck = &hcconf.HealthCheck{}
	}
	entryToSet.Entry.Fqdn = c.FQDN.String()
	entryToSetOrig := proto.Clone(entryToSet).(*gslbsvc.SetEntryRequest)
	var previousEntry *gslbsvc.SetEntryRequest
	resp, err := c.client.GetEntry(context.Background(), &gslbsvc.GetEntryRequest{
		Fqdn: entryToSet.GetEntry().GetFqdn(),
	})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	if err == nil {
		previousEntry = &gslbsvc.SetEntryRequest{
			Entry:       resp.GetEntry(),
			Healthcheck: resp.GetHealthcheck(),
		}
		proto.Merge(entryToSet, previousEntry)
	}
	if loaded {
		// override after merge
		entryToSet.Healthcheck.EnableTls = entryToSetOrig.Healthcheck.EnableTls
		return c.apply(previousEntry, entryToSet)
	}
	newEntryToSet, err := c.makeEntry()
	if err != nil {
		return err
	}
	proto.Merge(entryToSet, newEntryToSet)
	// override after merge
	entryToSet.Healthcheck.EnableTls = entryToSetOrig.Healthcheck.EnableTls
	return c.apply(previousEntry, entryToSet)
}

func (c *SetEntry) apply(previousEntry, currentEntry *gslbsvc.SetEntryRequest) error {
	confirm, err := DiffAndConfirm(previousEntry, currentEntry, c.Force)
	if err != nil {
		return err
	}
	if !confirm {
		return nil
	}
	_, err = c.client.SetEntry(context.Background(), currentEntry)
	if err != nil {
		return err
	}
	msg.Successf("Entry %s set successfully.", msg.Cyan(c.FQDN))
	return nil
}

func (c *SetEntry) makeEntry() (*gslbsvc.SetEntryRequest, error) {
	membersIpv4, err := ListMapToMembers(c.MembersIPv4)
	if err != nil {
		return nil, fmt.Errorf("invalid members ipv4: %s", err)
	}
	membersIpv6, err := ListMapToMembers(c.MembersIPv6)
	if err != nil {
		return nil, fmt.Errorf("invalid members ipv6: %s", err)
	}
	hc, err := c.makeHealthcheck()
	if err != nil {
		return nil, err
	}
	req := &gslbsvc.SetEntryRequest{
		Entry: &entries.Entry{
			Fqdn:              c.FQDN.String(),
			LbAlgoPreferred:   entries.LBAlgo(entries.LBAlgo_value[c.LBAlgoPreferred]),
			LbAlgoAlternate:   entries.LBAlgo(entries.LBAlgo_value[c.LBAlgoAlternate]),
			LbAlgoFallback:    entries.LBAlgo(entries.LBAlgo_value[c.LBAlgoFallback]),
			MaxAnswerReturned: c.MaxAnswerReturned,
			MembersIpv4:       membersIpv4,
			MembersIpv6:       membersIpv6,
			Ttl:               c.TTL,
			Permissions:       nil,
			Tags:              c.Tags,
		},
		Healthcheck: hc,
	}
	return req, nil
}

func (c *SetEntry) makeHealthcheck() (*hcconf.HealthCheck, error) {
	timeout, err := time.ParseDuration(c.HcTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout: %s", err)
	}
	interval, err := time.ParseDuration(c.HcInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid interval: %s", err)
	}
	hc := &hcconf.HealthCheck{
		Timeout:       durationpb.New(timeout),
		Interval:      durationpb.New(interval),
		Port:          c.HcPort,
		HealthChecker: nil,
		EnableTls:     c.HcEnableTls,
	}
	switch c.HcType {
	case "HTTP":
		rnge := &gsloctype.Int64Range{}
		if c.HttpCode != 0 {
			rnge.Start = int64(c.HttpCode)
			rnge.End = int64(c.HttpCode)
		}
		if c.HttpCodeRange != "" {
			splitRange := strings.Split(c.HttpCodeRange, "-")
			if len(splitRange) != 2 {
				return nil, fmt.Errorf("invalid http code range")
			}
			start, err := strconv.Atoi(splitRange[0])
			if err != nil {
				return nil, fmt.Errorf("invalid http code range: %s", err)
			}
			end, err := strconv.Atoi(splitRange[1])
			if err != nil {
				return nil, fmt.Errorf("invalid http code range: %s", err)
			}
			rnge.Start = int64(start)
			rnge.End = int64(end)
		}
		if rnge.Start == 0 && rnge.End == 0 {
			rnge.Start = int64(200)
			rnge.End = int64(200)
		}
		headers := make([]*core.HeaderValueOption, 0)
		for k, v := range c.HttpHeaders {
			headers = append(headers, &core.HeaderValueOption{
				Header: &core.HeaderValue{
					Key:   k,
					Value: v,
				},
				Append: false,
			})
		}
		hc.HealthChecker = &hcconf.HealthCheck_HttpHealthCheck{
			HttpHealthCheck: &hcconf.HttpHealthCheck{
				Host:                c.HttpHost,
				Path:                c.HttpPath,
				Send:                MakePayloadFromString(c.HttpSendPayload),
				Receive:             MakePayloadFromString(c.HttpReceivePayload),
				RequestHeadersToAdd: headers,
				ExpectedStatuses:    rnge,
				CodecClientType:     gsloctype.CodecClientType(gsloctype.CodecClientType_value[c.HttpCodecClientType]),
				Method:              hcconf.RequestMethod(hcconf.RequestMethod_value[c.HttpMethod]),
			},
		}
	case "TCP":
		payloads := make([]*hcconf.HealthCheckPayload, 0)
		for _, p := range c.TcpReceivePayloads {
			payloads = append(payloads, MakePayloadFromString(p))
		}
		hc.HealthChecker = &hcconf.HealthCheck_TcpHealthCheck{
			TcpHealthCheck: &hcconf.TcpHealthCheck{
				Send:    MakePayloadFromString(c.TcpSendPayload),
				Receive: payloads,
			},
		}
	case "GRPC":
		hc.HealthChecker = &hcconf.HealthCheck_GrpcHealthCheck{
			GrpcHealthCheck: &hcconf.GrpcHealthCheck{
				ServiceName: c.GRPCServiceName,
				Authority:   c.GRPCAuthority,
			},
		}
	default:
		hc.HealthChecker = &hcconf.HealthCheck_NoHealthCheck{
			NoHealthCheck: &hcconf.NoHealthCheck{},
		}
	}
	return hc, nil
}

func init() {
	desc := "Create or update an entry."
	cmd, err := parser.AddCommand(
		"set-entry",
		desc,
		desc,
		&setEntry)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"se"}
}
