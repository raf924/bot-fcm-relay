package fcm

import pkg "github.com/raf924/bot-grpc-relay"

type RelayConfig struct {
	StorageBucket    string               `yaml:"storageBucket"`
	ProjectId        string               `yaml:"projectId"`
	ServiceAccountId string               `yaml:"serviceAccountId"`
	CredentialsFile  string               `yaml:"credentialsFile"`
	Topic            string               `yaml:"topic"`
	ServerName       string               `yaml:"serverName"`
	Grpc             pkg.GrpcServerConfig `yaml:"grpc"`
	DatabaseURL      string               `yaml:"databaseUrl"`
}
