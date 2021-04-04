package fcm

import pkg "github.com/raf924/bot-grpc-relay"

type fcmRelayConfig struct {
	StorageBucket    string              `yaml:"storageBucket"`
	ProjectId        string              `yaml:"projectId"`
	ServiceAccountId string              `yaml:"serviceAccountId"`
	CredentialsFile  string              `yaml:"credentialsFile"`
	Topic            string              `yaml:"topic"`
	Grpc             pkg.GrpcRelayConfig `yaml:"grpc"`
	DatabaseURL      string              `yaml:"databaseUrl"`
}
