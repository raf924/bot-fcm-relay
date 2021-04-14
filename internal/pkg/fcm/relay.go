package fcm

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/base64"
	"errors"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"firebase.google.com/go/v4/messaging"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/raf924/bot-grpc-relay/pkg/server"
	"github.com/raf924/bot/pkg/queue"
	"github.com/raf924/connector-api/pkg/gen"
	"google.golang.org/api/option"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v2"
	"log"
	"net"
)

type fcmRelay struct {
	gen.UnimplementedConnectorServer
	users             []*gen.User
	user              *gen.User
	app               *firebase.App
	client            *messaging.Client
	token             string
	config            RelayConfig
	store             *firestore.Client
	db                *db.Client
	topic             string
	messageConsumer   *queue.Consumer
	messageProducer   *queue.Producer
	connectorExchange *queue.Exchange
}

func (f *fcmRelay) Register(ctx context.Context, _ *gen.RegistrationPacket) (*gen.ConfirmationPacket, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("no metadata")
	}
	values := md.Get("topic")
	if len(values) == 0 {
		return nil, errors.New("topic metadata must have a value")
	}
	f.topic = values[0]
	log.Println("Topic is", f.topic)
	return &gen.ConfirmationPacket{
		BotUser: f.user,
		Users:   f.users,
	}, nil
}

func (f *fcmRelay) ReadMessages(_ *empty.Empty, _ gen.Connector_ReadMessagesServer) error {
	return nil
}

func (f *fcmRelay) ReadCommands(_ *empty.Empty, _ gen.Connector_ReadCommandsServer) error {
	return nil
}

func (f *fcmRelay) ReadUserEvents(_ *empty.Empty, _ gen.Connector_ReadUserEventsServer) error {
	return nil
}

func (f *fcmRelay) SendMessage(_ context.Context, packet *gen.BotPacket) (*empty.Empty, error) {
	return &empty.Empty{}, f.messageProducer.Produce(packet)
}

func NewFCMRelay(config interface{}, connectorExchange *queue.Exchange) *fcmRelay {
	data, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}
	var conf RelayConfig
	if err := yaml.Unmarshal(data, &conf); err != nil {
		panic(err)
	}
	return newFCMRelay(conf, connectorExchange)
}

func newFCMRelay(config RelayConfig, connectorExchange *queue.Exchange) *fcmRelay {
	return &fcmRelay{
		config:            config,
		connectorExchange: connectorExchange,
	}
}

func (f *fcmRelay) Ping(context.Context, *empty.Empty) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (f *fcmRelay) Start(botUser *gen.User, users []*gen.User, _ string) error {
	/*if !f.config.Grpc.TLS.Enabled {
		return errors.New("can't start: TLS must be enabled")
	}*/
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
	f.client, err = f.app.Messaging(context.TODO())
	if err != nil {
		return err
	}
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", f.config.Grpc.Port))
	if err != nil {
		return err
	}
	if len(f.config.ServerName) > 0 {
		err := CheckPairing(l, f.config)
		if err != nil {
			return err
		}
		err = UpdateFirestore(f.store, f.config.ServerName)
		if err != nil {
			return err
		}
	}
	mQueue := queue.NewQueue()
	f.messageProducer, _ = mQueue.NewProducer()
	f.messageConsumer, _ = mQueue.NewConsumer()
	return server.StartServer(l, f, f.config.Grpc)
}

func (f *fcmRelay) send(payloadType string, v proto.Message) error {
	payload, err := proto.Marshal(v)
	if err != nil {
		return err
	}
	log.Println("Sending")
	_, err = f.client.Send(
		context.TODO(),
		&messaging.Message{Topic: f.topic, Data: map[string]string{
			"type":    payloadType,
			"payload": base64.StdEncoding.EncodeToString(payload),
		}},
	)
	return err
}

func (f *fcmRelay) Send(message proto.Message) error {
	var typ string
	switch message.(type) {
	case *gen.CommandPacket:
		typ = "command"
	case *gen.MessagePacket:
		typ = "message"
	case *gen.UserPacket:
		typ = "userEvent"
	default:
		log.Println("unknown packet type")
		return nil
	}
	return f.send(typ, message)
}

func (f *fcmRelay) Recv() (*gen.BotPacket, error) {
	p, err := f.messageConsumer.Consume()
	if err != nil {
		return nil, errors.New("no message to read")
	}
	return p.(*gen.BotPacket), nil
}

func (f *fcmRelay) Commands() []*gen.Command {
	return nil
}
