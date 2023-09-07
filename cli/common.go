package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	msg "github.com/ArthurHlt/messages"
	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"
	"github.com/mitchellh/mapstructure"
	"github.com/olekukonko/tablewriter"
	"github.com/orange-cloudfoundry/gsloc-cli/highlight"
	"github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/api/config/entries/v1"
	hcconf "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/api/config/healthchecks/v1"
	"github.com/orange-cloudfoundry/gsloc-go-sdk/helpers"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"path/filepath"
	"regexp"
	kyaml "sigs.k8s.io/yaml"
	"strings"
)

var emptyJsonRegex = regexp.MustCompile(`\{\s*\}`)

//var revertEnumType = map[string]string{
//	helpers.TypeUrl[*cert.Certificate]():             "certificate",
//	helpers.TypeUrl[*cluster.Cluster]():              "cluster",
//	helpers.TypeUrl[*instance.Instance]():            "instance",
//	helpers.TypeUrl[*route.Route]():                  "route",
//	helpers.TypeUrl[*vip.VIP]():                      "vip",
//	helpers.TypeUrl[*vs.VirtualServer]():             "vs",
//	helpers.TypeUrl[*cluster.ClusterEndpointStats](): "endpoints",
//}

func ListMapToMembers(lm []map[string]string) ([]*entries.Member, error) {
	var members []*entries.Member
	for _, m := range lm {
		member, err := MapToMember(m)
		if err != nil {
			if member.Ip != "" {
				return nil, fmt.Errorf("failed to parse member %s: %w", member.Ip, err)
			}
			return nil, err
		}
		members = append(members, member)
	}
	return members, nil
}

func MapToMember(m map[string]string) (*entries.Member, error) {
	var mm MemberMap
	err := mapstructure.WeakDecode(m, &mm)
	if err != nil {
		return nil, err
	}
	member := &entries.Member{
		Ip:       mm.Ip,
		Ratio:    uint32(mm.Ratio),
		Dc:       mm.DC,
		Disabled: mm.Disabled,
	}
	err = member.Validate()
	if err != nil {
		return member, err
	}
	return member, nil
}

func FileToProto[T proto.Message](file string) (protoMsg T, loaded bool, err error) {
	var protoMsgDef T
	protoMsg = protoMsgDef.ProtoReflect().New().Interface().(T)
	if file == "" {
		return protoMsg, false, nil
	}
	content, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return protoMsg, false, nil
		}
		return protoMsg, false, fmt.Errorf("failed to read file %s: %w", file, err)
	}

	ext := strings.ToLower(filepath.Ext(file))
	if ext == ".yaml" || ext == ".yml" {
		content, err = kyaml.YAMLToJSON(content)
		if err != nil {
			return protoMsg, false, fmt.Errorf("failed to convert yaml to json file %s: %w", file, err)
		}
	}

	if strings.TrimSpace(string(content)) == "" || string(content) == "null" {
		return protoMsg, false, nil
	}

	if emptyJsonRegex.Match(content) {
		return protoMsg, false, nil
	}

	err = protojson.Unmarshal(content, protoMsg)
	if err != nil {
		return protoMsg, false, fmt.Errorf("failed to unmarshal json file %s: %w", file, err)
	}
	return protoMsg, true, nil
}

func ProtoToYaml(pMsg proto.Message) ([]byte, error) {
	data, err := protojson.MarshalOptions{
		Multiline:       true,
		UseProtoNames:   true,
		Indent:          "  ",
		EmitUnpopulated: true,
	}.Marshal(pMsg)
	if err != nil {
		return nil, err
	}
	mapProto := make(map[string]any)
	err = json.Unmarshal(data, &mapProto)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(mapProto)
}

func MakePayloadFromString(txt string) *hcconf.HealthCheckPayload {
	if txt == "" {
		return nil
	}
	return &hcconf.HealthCheckPayload{
		Payload: &hcconf.HealthCheckPayload_Text{
			Text: txt,
		},
	}
}

func NameFromAny(anyMsg *anypb.Any) string {
	data, err := protojson.MarshalOptions{
		Multiline:       true,
		UseProtoNames:   true,
		Indent:          "  ",
		EmitUnpopulated: true,
	}.Marshal(anyMsg)
	if err != nil {
		panic(err)
	}
	mapProto := make(map[string]any)
	err = json.Unmarshal(data, &mapProto)
	if err != nil {
		panic(err)
	}
	name, ok := mapProto["name"]
	if !ok {
		return ""
	}
	return fmt.Sprint(name)
}

func ProtoDiffReport(from, dest proto.Message) (dyff.Report, error) {
	var fromYaml, destYaml []byte
	var err error
	if from != nil {
		fromYaml, err = ProtoToYaml(from)
		if err != nil {
			return dyff.Report{}, err
		}
	} else {
		fromYaml = []byte("")
	}
	if dest != nil {
		destYaml, err = ProtoToYaml(dest)
		if err != nil {
			return dyff.Report{}, err
		}
	} else {
		destYaml = []byte("")
	}

	fromDoc, err := ytbx.LoadDocuments(fromYaml)
	if err != nil {
		return dyff.Report{}, fmt.Errorf("failed to load input files: %s", err.Error())
	}
	toDoc, err := ytbx.LoadDocuments(destYaml)
	if err != nil {
		return dyff.Report{}, fmt.Errorf("failed to load input files: %s", err.Error())
	}

	iFrom := ytbx.InputFile{
		Documents: fromDoc,
		Names:     []string{helpers.GetIdentifier(from)},
	}
	iTo := ytbx.InputFile{
		Documents: toDoc,
		Names:     []string{helpers.GetIdentifier(dest)},
	}

	return dyff.CompareInputFiles(iFrom, iTo,
		dyff.IgnoreOrderChanges(true),
	)
}

func PrintProtoDiff(from, dest proto.Message) error {
	report, err := ProtoDiffReport(from, dest)
	if err != nil {
		return fmt.Errorf("failed to compare input files: %s", err.Error())
	}

	reportWriter := &dyff.HumanReport{
		Report:               report,
		NoTableStyle:         true,
		OmitHeader:           true,
		UseGoPatchPaths:      false,
		MinorChangeThreshold: 0.1,
	}
	return reportWriter.WriteReport(msg.Output())
}

func ProtoDiffContent(from, dest proto.Message) (string, error) {
	report, err := ProtoDiffReport(from, dest)
	if err != nil {
		return "", fmt.Errorf("failed to compare input files: %s", err.Error())
	}

	reportWriter := &dyff.HumanReport{
		Report:               report,
		NoTableStyle:         true,
		OmitHeader:           true,
		UseGoPatchPaths:      false,
		MinorChangeThreshold: 0.1,
	}
	buf := &bytes.Buffer{}

	err = reportWriter.WriteReport(buf)
	if err != nil {
		return "", fmt.Errorf("failed to write report: %s", err.Error())
	}
	return buf.String(), nil
}

func PrintProtoHuman(pMsg proto.Message) error {
	dataYml, err := ProtoToYaml(pMsg)
	if err != nil {
		return fmt.Errorf("failed to convert proto to yaml: %w", err)
	}
	result, err := highlight.Highlight(bytes.NewBuffer(dataYml))
	if err != nil {
		return fmt.Errorf("failed to highlight yaml: %w", err)
	}
	fmt.Println(result)
	return nil
}

func PrintProtoJson(pMsg proto.Message) error {
	data, err := protojson.MarshalOptions{
		Multiline:       true,
		Indent:          "  ",
		UseProtoNames:   true,
		EmitUnpopulated: true,
	}.Marshal(pMsg)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func PrintProtoListJson[T proto.Message](msgs []T) error {
	data, err := helpers.MarshalListProtoMessage[T](protojson.MarshalOptions{
		Multiline:       true,
		Indent:          "  ",
		UseProtoNames:   true,
		EmitUnpopulated: true,
	}, msgs)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func MakeTableWriter(headers []string, writers ...io.Writer) *tablewriter.Table {
	writer := msg.Output()
	if len(writers) > 0 {
		writer = writers[0]
	}
	table := tablewriter.NewWriter(writer)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	return table
}

func DiffAndConfirm(from, dest proto.Message, force bool) (bool, error) {
	msg.Info("Change to be made:")
	msg.Printf("━━━━━\n")
	err := PrintProtoDiff(from, dest)
	if err != nil {
		return false, err
	}
	if force {
		return true, nil
	}
	confirm := false
	prompt := &survey.Confirm{
		Message: "Do you confirm theses changes?",
	}
	err = survey.AskOne(prompt, &confirm)
	if err != nil {
		return false, err
	}
	return confirm, nil
}

// IsFqdn checks if a domain name is fully qualified.
func IsFqdn(s string) bool {
	// Check for (and remove) a trailing dot, returning if there isn't one.
	if s == "" || s[len(s)-1] != '.' {
		return false
	}
	s = s[:len(s)-1]

	// If we don't have an escape sequence before the final dot, we know it's
	// fully qualified and can return here.
	if s == "" || s[len(s)-1] != '\\' {
		return true
	}

	// Otherwise we have to check if the dot is escaped or not by checking if
	// there are an odd or even number of escape sequences before the dot.
	i := strings.LastIndexFunc(s, func(r rune) bool {
		return r != '\\'
	})
	return (len(s)-i)%2 != 0
}

// Fqdn return the fully qualified domain name from s.
// If s is already fully qualified, it behaves as the identity function.
func Fqdn(s string) string {
	if IsFqdn(s) {
		return s
	}
	return s + "."
}
