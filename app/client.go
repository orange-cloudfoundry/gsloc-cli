package app

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	gslbsvc "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/services/gslb/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"os"
	"os/user"
	"path/filepath"
)

type gslocConfig struct {
	Host       string `json:"host"`
	Username   string `json:"username"`
	SkipVerify bool   `json:"skip_verify"`
}

type Client struct {
}

func makeGrpcConn(host, username, password string, skipVerify bool, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: skipVerify,
	})))
	//loginSvc, err := svclogin.NewLoginServiceWithStoreDir(host, username, password, os.TempDir(), opts...)
	//if err != nil {
	//	return nil, fmt.Errorf("Error when creating grpc connection: %s", err)
	//}
	//_, err = loginSvc.GetRequestMetadata(context.Background())
	//if err != nil {
	//	return nil, err
	//}
	//return grpc.Dial(host, append(opts, grpc.WithPerRPCCredentials(loginSvc))...)
	return grpc.Dial(host, opts...)
}

func retrieveConfig(path string) (*gslocConfig, error) {
	confRaw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found, please login first")
		}
		return nil, err
	}
	config := &gslocConfig{}
	err = json.Unmarshal(confRaw, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func GetCurrentUsername(path string) string {
	config, err := retrieveConfig(path)
	if err != nil {
		me, err := user.Current()
		if err != nil {
			return ""
		}
		return me.Username
	}
	return config.Username
}

func CreateConnFromFile(path string) (*grpc.ClientConn, error) {
	config, err := retrieveConfig(path)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("error when unmarshalling config file: %w", err)
	}
	return makeGrpcConn(config.Host, config.Username, "", config.SkipVerify)
}

func CreateConn(path string, host, username, password string, skipVerify bool) (*grpc.ClientConn, error) {

	config, err := retrieveConfig(path)
	if err == nil && host == "" {
		host = config.Host
		skipVerify = config.SkipVerify
	}

	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return nil, err
	}
	conn, err := makeGrpcConn(host, username, password, skipVerify)
	if err != nil {
		return nil, err
	}
	// nolint:errcheck
	b, _ := json.MarshalIndent(gslocConfig{
		Host:       host,
		Username:   username,
		SkipVerify: skipVerify,
	}, "", "  ")
	err = os.WriteFile(path, b, 0644)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func MakeClient(conn *grpc.ClientConn) gslbsvc.GSLBClient {

	return gslbsvc.NewGSLBClient(conn)
}
