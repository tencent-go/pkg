package wsx

import (
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/util"
	"github.com/lxzan/gws"
	"github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v5"
)

func Broadcast[T any](channel EventChannelBuilder[T], connections []Conn, data T) {
	payload, err := util.Msgpack().Marshal(sendMsgWrapper{Topic: channel.Topic(), Data: data})
	if err != nil {
		logrus.WithError(err).Error("marshal msgpack fail")
		return
	}
	var b = gws.NewBroadcaster(gws.OpcodeBinary, payload)
	defer func() { _ = b.Close() }()
	for _, item := range connections {
		conn, ok := item.(*connWrapper)
		if !ok {
			logrus.Error("conn to connWrapper failed")
			continue
		}
		_ = b.Broadcast(conn.Conn)
	}
}

func SendMessage[T any](channel EventChannelBuilder[T], connection Conn, data T) errx.Error {
	if !connection.Subscriptions().Exists(channel.Topic()) {
		return errx.Newf("topic %s undescribe", channel.Topic())
	}
	return connection.Send(channel.Topic(), data)
}

type receiveMsgWrapper struct {
	Topic string             `json:"topic"`
	Data  msgpack.RawMessage `json:"data"`
}

type sendMsgWrapper struct {
	Topic string `json:"topic"`
	Data  any    `json:"data"`
}

type SubscribeTopicsEvent struct {
	Topics []string `json:"topics"`
}

type ErrorEvent struct {
	Message string `json:"message"`
}
