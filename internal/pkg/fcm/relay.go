package fcm

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"errors"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	_ "github.com/raf924/bot-grpc-relay"
	pkg "github.com/raf924/bot-grpc-relay"
	"github.com/raf924/bot/pkg/relay"
	"github.com/raf924/connector-api/pkg/gen"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v2"
)

type fcmRelay struct {
	users       []*gen.User
	user        *gen.User
	app         *firebase.App
	client      *messaging.Client
	token       string
	config      fcmRelayConfig
	sourceRelay relay.BotRelay
	store       *firestore.Client
}

func NewFCMRelay(config interface{}) relay.BotRelay {
	data, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}
	var conf fcmRelayConfig
	if err := yaml.Unmarshal(data, &conf); err != nil {
		panic(err)
	}
	return &fcmRelay{
		config:      conf,
		sourceRelay: pkg.NewGrpcBotRelay(conf.grpc),
	}
}

func (f *fcmRelay) Start(botUser *gen.User, users []*gen.User) error {
	f.user = botUser
	f.users = users
	var err error
	f.app, err = firebase.NewApp(context.TODO(), &firebase.Config{
		StorageBucket:    f.config.storageBucket,
		ProjectID:        f.config.projectId,
		ServiceAccountID: f.config.serviceAccountId,
	}, option.WithCredentialsFile(f.config.credentialsFile))
	if err != nil {
		return err
	}
	f.store, err = f.app.Firestore(context.TODO())
	if err != nil {
		return err
	}
	_, err = f.store.Collection("/topics").Where("topic", "==", f.config.topic).Limit(1).Documents(context.TODO()).Next()
	if errors.Is(err, iterator.Done) {
		_, err = f.store.Collection("/topics").NewDoc().Update(context.TODO(), []firestore.Update{{Path: "topic", Value: f.config.topic}})
	}
	if err != nil {
		return err
	}
	f.client, err = f.app.Messaging(context.TODO())
	if err != nil {
		return err
	}
	return f.sourceRelay.Start(botUser, users)
}

func (f *fcmRelay) send(payloadType string, v interface{}) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = f.client.Send(context.TODO(), &messaging.Message{Topic: f.config.topic, Data: map[string]string{"type": payloadType, "payload": string(payload)}})
	return err
}

func (f *fcmRelay) PassMessage(message *gen.MessagePacket) error {
	return f.send("message", message)
}

func (f *fcmRelay) PassEvent(event *gen.UserPacket) error {
	return f.send("event", event)
}

func (f *fcmRelay) PassCommand(command *gen.CommandPacket) error {
	return f.send("command", command)
}

func (f *fcmRelay) RecvMsg(packet *gen.BotPacket) error {
	return f.sourceRelay.RecvMsg(packet)
}

func (f *fcmRelay) Trigger() string {
	return ""
}

func (f *fcmRelay) Commands() []*gen.Command {
	return nil
}

func (f *fcmRelay) Ready() <-chan struct{} {
	return f.sourceRelay.Ready()
}
