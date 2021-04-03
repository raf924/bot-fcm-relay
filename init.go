package bot_fcm_relay

import (
	"github.com/raf924/bot-fcm-relay/internal/pkg/fcm"
	"github.com/raf924/bot/pkg/relay"
)

func init() {
	relay.RegisterBotRelay("fcm", fcm.NewFCMRelay)
}
