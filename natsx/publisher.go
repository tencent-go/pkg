package natsx

import (
	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/util"
	"github.com/nats-io/nats.go"
)

type Publisher[T any] interface {
	Publish(ctx ctxx.Context, msg T) errx.Error
}

func newPublisher[T any](nc *nats.Conn, subject string, args ...string) (*publisher[T], errx.Error) {
	sub, missingPlaceholders := replaceSubjectPlaceholders(subject, args...)
	if len(missingPlaceholders) > 0 {
		return nil, errx.Newf("missing placeholders: %v", missingPlaceholders)
	}
	return &publisher[T]{nc, sub}, nil
}

type publisher[T any] struct {
	nc      *nats.Conn
	subject string
}

func (p *publisher[T]) Publish(ctx ctxx.Context, msg T) errx.Error {
	data, err := util.Json().Marshal(msg)
	if err != nil {
		return err
	}
	natsMsg := &nats.Msg{
		Subject: p.subject,
		Data:    data,
		Header:  newNatsHeader(ctx),
	}
	if msgIdGetter, ok := any(msg).(MsgIdGetter); ok {
		msgId := msgIdGetter.MsgID()
		if msgId != "" {
			natsMsg.Header.Set("Nats-Msg-Id", msgId)
		}
	}
	e := p.nc.PublishMsg(natsMsg)
	if e != nil {
		return errx.Wrap(e).AppendMsgf("subject %s publish message failed", p.subject).Err()
	}
	return nil
}
