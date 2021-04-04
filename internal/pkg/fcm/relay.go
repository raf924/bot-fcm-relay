package fcm

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/base64"
	"errors"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"firebase.google.com/go/v4/messaging"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/raf924/bot-grpc-relay/pkg/server"
	"github.com/raf924/bot/pkg/relay"
	"github.com/raf924/connector-api/pkg/gen"
	"google.golang.org/api/option"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v2"
)

type grpcRelay interface {
	relay.BotRelay
	gen.ConnectorServer
}

type fcmRelay struct {
	gen.UnimplementedConnectorServer
	users  []*gen.User
	user   *gen.User
	app    *firebase.App
	client *messaging.Client
	token  string
	config fcmRelayConfig
	//sourceRelay grpcRelay
	store        *firestore.Client
	db           *db.Client
	topic        string
	messageQueue MessageQueue
}

func (f *fcmRelay) Register(ctx context.Context, packet *gen.RegistrationPacket) (*gen.ConfirmationPacket, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("no metadata")
	}
	values := md.Get("topic")
	if len(values) == 0 {
		return nil, errors.New("topic metadata must have a value")
	}
	f.topic = values[0]
	println("Topic is", f.topic)
	return &gen.ConfirmationPacket{
		BotUser: f.user,
		Users:   f.users,
	}, nil
}

func (f *fcmRelay) ReadMessages(empty *empty.Empty, server gen.Connector_ReadMessagesServer) error {
	return nil
}

func (f *fcmRelay) ReadCommands(empty *empty.Empty, server gen.Connector_ReadCommandsServer) error {
	return nil
}

func (f *fcmRelay) ReadUserEvents(empty *empty.Empty, server gen.Connector_ReadUserEventsServer) error {
	return nil
}

func (f *fcmRelay) SendMessage(ctx context.Context, packet *gen.BotPacket) (*empty.Empty, error) {
	f.messageQueue.Push(packet)
	return &empty.Empty{}, nil
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
		config:       conf,
		messageQueue: NewMessageQueue(),
		//sourceRelay: pkg.NewGrpcBotRelay(conf.Grpc),
	}
}

func (f *fcmRelay) Start(botUser *gen.User, users []*gen.User) error {
	f.user = botUser
	f.users = users
	var err error
	f.app, err = firebase.NewApp(context.TODO(), &firebase.Config{
		StorageBucket:    f.config.StorageBucket,
		ProjectID:        f.config.ProjectId,
		ServiceAccountID: f.config.ServiceAccountId,
		DatabaseURL:      f.config.DatabaseURL,
	}, option.WithCredentialsFile(f.config.CredentialsFile))
	if err != nil {
		return err
	}
	f.store, err = f.app.Firestore(context.TODO())
	if err != nil {
		return err
	}
	f.db, err = f.app.Database(context.TODO())
	if err != nil {
		return err
	}
	f.client, err = f.app.Messaging(context.TODO())
	if err != nil {
		return err
	}
	return server.StartServer(f, f.config.Grpc)
}

func (f *fcmRelay) send(payloadType string, v proto.Message) error {
	payload, err := proto.Marshal(v)
	if err != nil {
		return err
	}
	_, err = f.client.Send(
		context.TODO(),
		&messaging.Message{Topic: f.topic, Data: map[string]string{
			"type":    payloadType,
			"payload": base64.StdEncoding.EncodeToString(payload),
		}},
	)
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
	p := f.messageQueue.Pop()
	if p == nil {
		return errors.New("no message to read")
	}
	packet.Message = p.Message
	packet.Private = p.Private
	packet.Recipient = p.Recipient
	packet.Timestamp = p.Timestamp
	println("Received message")
	return nil
}

func (f *fcmRelay) Trigger() string {
	return "!"
}

func (f *fcmRelay) Commands() []*gen.Command {
	return nil
}

func (f *fcmRelay) Ready() <-chan struct{} {
	c := make(chan struct{})
	go func() {
		c <- struct{}{}
	}()
	return c
}
