package bot_fcm_relay

import (
	"github.com/raf924/bot-fcm-relay/internal/pkg/fcm"
	"github.com/raf924/bot/pkg/queue"
	"github.com/raf924/bot/pkg/relay/server"
)

func init() {
	server.RegisterRelayServer("fcm", func(config interface{}, connectorExchange *queue.Exchange) server.RelayServer {
		return fcm.NewFCMRelay(config, connectorExchange)
	})
}

type FCMRelayConfig = fcm.RelayConfig
