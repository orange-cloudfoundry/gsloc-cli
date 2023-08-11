package cli

import (
	"context"
	"fmt"
	msg "github.com/ArthurHlt/messages"
	"github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/api/config/core/v1"
	hcconf "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/api/config/healthchecks/v1"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
	gsloctype "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/type/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"strconv"
	"strings"
	"time"
)

type SetHealthcheck struct {
	FQDN        *FQDN  `positional-args:"true" positional-arg-name:"'fqdn'" required:"true"`
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

var setHealthcheck SetHealthcheck

func (c *SetHealthcheck) SetClient(client gslbsvc.GSLBClient) {
	c.client = client
}

func (c *SetHealthcheck) Execute([]string) error {
	hc, err := c.makeHealthcheck()
	if err != nil {
		return err
	}
	setHcReq := &gslbsvc.SetHealthCheckRequest{
		Fqdn:        c.FQDN.String(),
		Healthcheck: hc,
	}

	var previousEntry *gslbsvc.SetHealthCheckRequest
	resp, err := c.client.GetHealthCheck(context.Background(), &gslbsvc.GetHealthCheckRequest{
		Fqdn: c.FQDN.String(),
	})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	if err == nil {
		previousEntry = &gslbsvc.SetHealthCheckRequest{
			Fqdn:        c.FQDN.String(),
			Healthcheck: resp.Healthcheck,
		}
		proto.Merge(setHcReq, previousEntry)
	}
	confirm, err := DiffAndConfirm(previousEntry, setHcReq, c.Force)
	if err != nil {
		return err
	}
	if !confirm {
		return nil
	}
	_, err = c.client.SetHealthCheck(context.Background(), setHcReq)
	if err != nil {
		return err
	}
	msg.Successf("Healthcheck for %s set successfully.", msg.Cyan(c.FQDN))
	return nil
}

func (c *SetHealthcheck) makeHealthcheck() (*hcconf.HealthCheck, error) {
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
		&setHealthcheck)
	if err != nil {
		panic(err)
	}
	cmd.Aliases = []string{"se"}
}
