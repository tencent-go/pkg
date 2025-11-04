package wsdoc

import (
	"github.com/tencent-go/pkg/doc/schema"
	"github.com/tencent-go/pkg/util"
	"github.com/tencent-go/pkg/wsx"
)

type EventChannel struct {
	Topic       string
	Type        *schema.Type
	Description string
}

func NewGroups(schemaCollection schema.Collection, channels []wsx.EventChannel) []EventChannel {
	res := make([]EventChannel, len(channels))
	for i, channel := range channels {
		typ, _ := schemaCollection.ParseAndGetType(channel.MessageType(), util.TagJson)
		res[i] = EventChannel{
			Topic: channel.Topic(),
			Type:  typ,
		}
	}
	return res
}
