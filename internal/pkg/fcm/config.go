package fcm

import pkg "github.com/raf924/bot-grpc-relay"

type fcmRelayConfig struct {
	storageBucket    string `yaml:"storageBucket"`
	projectId        string `yaml:"projectId"`
	serviceAccountId string `yaml:"serviceAccountId"`
	senderId         string `yaml:"senderId"`
	credentialsFile  string `yaml:"credentialsFile"`
	serverKeyFile    string `yaml:"serverKeyFile"`
	topic            string `yaml:"topic"`
	grpc             pkg.GrpcRelayConfig
}
