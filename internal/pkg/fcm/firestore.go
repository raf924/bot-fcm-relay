package fcm

import (
	"cloud.google.com/go/firestore"
	"context"
)

type Host struct {
	ServerName string `firestore:"serverName"`
	Name       string `firestore:"name"`
}

func UpdateFirestore(client *firestore.Client, serverName string) error {
	serverHost := Host{
		ServerName: serverName,
		Name:       "default",
	}
	hosts := client.Collection("hosts")
	q := hosts.Where("serverName", "==", serverName).Documents(context.Background())
	docs, err := q.GetAll()
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		_, err := hosts.NewDoc().Create(context.Background(), &serverHost)
		if err != nil {
			return err
		}
	}
	return nil
}
