package cli

import (
	"fmt"
	"github.com/ArthurHlt/go-flags"
	msg "github.com/ArthurHlt/messages"
	"github.com/mitchellh/go-homedir"
	"github.com/orange-cloudfoundry/gsloc-cli/app"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const defaultConfigPath = "~/.gsloc/config.json"

type Options struct {
	ConfigPath string `short:"c" long:"config" description:"Path to config file" default:"~/.gsloc/config.json" env:"GSLOC_CONFIG_PATH"`
	Version    func() `          long:"version" description:"Show version"`
}

type SetClient interface {
	SetClient(client gslbsvc.GSLBClient)
}

var opts = Options{}
var parser = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)

func Start(version, commit, date string) (err error) {
	askVersion := false
	opts.Version = func() {
		askVersion = true
		fmt.Printf("lbaas %s, commit %s, built at %s\n", version, commit, date)
	}
	var clientConn *grpc.ClientConn
	defer func() {
		if clientConn != nil {
			clientConn.Close() //nolint:errcheck
		}
	}()

	parser.CommandHandler = func(command flags.Commander, args []string) error {
		if cmd, ok := command.(SetClient); ok {

			clientConn, err = app.CreateConnFromFile(ExpandConfigPath())
			if err != nil {
				return err
			}
			cmd.SetClient(app.MakeClient(clientConn))
		}

		return command.Execute(args)
	}
	_, err = parser.Parse()
	if err != nil {
		errFlag, isErrFlag := err.(*flags.Error)
		if isErrFlag && askVersion && errFlag.Type == flags.ErrCommandRequired {
			return nil
		}
		if isErrFlag && errFlag.Type == flags.ErrHelp {
			msg.Print(err.Error()) // nolint:errcheck
			return nil
		}
		errStatus, ok := status.FromError(err)
		if ok {
			err = fmt.Errorf("Error Code: %s\nError Message: %s", msg.Yellow(errStatus.Code()), msg.Blue(errStatus.Message()))
		}
		return err
	}
	return nil
}

func ExpandConfigPath() string {
	cp, err := homedir.Expand(opts.ConfigPath)
	if err != nil {
		panic(fmt.Sprintf("Error while expanding config path %s", err.Error()))
	}
	return cp
}
