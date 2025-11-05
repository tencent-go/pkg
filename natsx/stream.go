package natsx

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/tencent-go/pkg/errx"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/sirupsen/logrus"
	"github.com/tencent-go/pkg/ctxx"
)

func NewStreamBuilder(streamName string) StreamBuilder {
	if streamName == "" {
		logrus.Panic("nats stream name is required")
	}
	if !nameRe.MatchString(streamName) {
		logrus.Panicf("nats stream name %s is invalid", streamName)
	}
	return &stream{
		streamOptions: streamOptions{name: streamName},
	}
}

type Stream interface {
	Conn() *nats.Conn
	Stream() (jetstream.Stream, errx.Error)
	JetStream() jetstream.JetStream
	Config() jetstream.StreamConfig
}

type StreamBuilder interface {
	Stream
	WithConn(conn *nats.Conn) StreamBuilder
	WithSubjects(subjects ...string) StreamBuilder
	WithConfig(config jetstream.StreamConfig) StreamBuilder
}

type streamOptions struct {
	name      string
	conn      *nats.Conn
	subjects  []string
	config    *jetstream.StreamConfig
	jetStream jetstream.JetStream
}

type stream struct {
	streamOptions
	stream jetstream.Stream
}

func (s *stream) WithConn(conn *nats.Conn) StreamBuilder {
	o := s.streamOptions
	o.conn = conn
	return &stream{streamOptions: o}
}

func (s *stream) WithSubjects(subjects ...string) StreamBuilder {
	o := s.streamOptions
	o.subjects = subjects
	return &stream{streamOptions: o}
}

func (s *stream) WithConfig(config jetstream.StreamConfig) StreamBuilder {
	o := s.streamOptions
	o.config = &config
	return &stream{streamOptions: o}
}

func (s *stream) getConfig() jetstream.StreamConfig {
	//TODO
	if s.config != nil {
		return *s.config
	}
	if len(s.subjects) == 0 {
		logrus.Panicf("stream %s has no subjects", s.name)
	}
	return jetstream.StreamConfig{
		Name:       s.name,
		Subjects:   s.subjects,
		Storage:    jetstream.FileStorage,
		Retention:  jetstream.InterestPolicy,
		MaxMsgs:    -1,
		MaxBytes:   -1,
		MaxAge:     0,
		Discard:    jetstream.DiscardOld,
		Duplicates: time.Hour,
	}
}

func (s *stream) Config() jetstream.StreamConfig {
	return s.getConfig()
}

func (s *stream) Stream() (jetstream.Stream, errx.Error) {
	if s.stream != nil {
		return s.stream, nil
	}
	js := s.JetStream()
	ctx := ctxx.Background()
	res, e := js.Stream(context.Background(), s.name)
	conf := s.getConfig()
	if e != nil {
		if !errors.Is(e, jetstream.ErrStreamNotFound) {
			return nil, errx.Wrap(e).AppendMsgf("failed to get stream %s: %v", s.name, e).Err()
		}
		res, e = js.CreateOrUpdateStream(ctx, conf)
		if e != nil {
			return nil, errx.Wrap(e).AppendMsgf("failed to create stream %s: %v", s.name, e).Err()
		}
		logrus.Infof("created stream %s", s.name)
		s.stream = res
		return res, nil
	}
	if !compareStreamConfig(res.CachedInfo().Config, conf) {
		res, e = js.UpdateStream(ctx, conf)
		if e != nil {
			return nil, errx.Wrap(e).AppendMsgf("failed to update stream %s: %v", s.name, e).Err()
		}
		logrus.Infof("updated stream %s", s.name)
	}
	s.stream = res
	return res, nil
}

func (s *stream) JetStream() jetstream.JetStream {
	if s.jetStream == nil {
		js, e := jetstream.New(s.Conn())
		if e != nil {
			logrus.Panicf("failed to create jetstream: %v", e)
		}
		s.jetStream = js
	}
	return s.jetStream
}

func (s *stream) Conn() *nats.Conn {
	if s.conn == nil {
		s.conn = getDefaultConn()
	}
	return s.conn
}

func compareStreamConfig(c1, c2 jetstream.StreamConfig) bool {
	return reflect.DeepEqual(c1, c2)
}
