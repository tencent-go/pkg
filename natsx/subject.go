package natsx

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/tencent-go/pkg/errx"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/sirupsen/logrus"
)

// NewSubjectBuilder 佔位符用`{}`,分隔符用`.` 例如`user.{userId}.created`
func NewSubjectBuilder[T any](subject string) SubjectBuilder[T] {
	if !validateSubject(subject) {
		logrus.Panicf("invalid subject %s", subject)
	}
	return &subjectBuilder[T]{subjectOptions: subjectOptions{subject: subject}}
}

type Subject[T any] interface {
	Conn() *nats.Conn
	Publisher() (Publisher[T], errx.Error)
	MustPublisher() Publisher[T]
	Subscriber() Subscriber[T]

	JetStream() jetstream.JetStream
	StreamPublisher() (StreamPublisher[T], errx.Error)
	MustStreamPublisher() StreamPublisher[T]

	EphemeralConsumer() (jetstream.Consumer, errx.Error)
	EphemeralStreamSubscriber() (StreamSubscriber[T], errx.Error)
	MustEphemeralStreamSubscriber() StreamSubscriber[T]

	DurableConsumer() (jetstream.Consumer, errx.Error)
	DurableStreamSubscriber() (StreamSubscriber[T], errx.Error)
	MustDurableStreamSubscriber() StreamSubscriber[T]
}

type SubjectBuilder[T any] interface {
	Subject[T]
	WithArgs(args ...string) SubjectBuilder[T]
	WithConn(conn *nats.Conn) SubjectBuilder[T]
	WithStream(stream Stream) SubjectBuilder[T]
	WithHandlerProcessTimeout(timeout time.Duration) SubjectBuilder[T] //默認為consumer config ack wait,僅durable subscribe有效
	WithConsumerConfig(consumerConfig jetstream.ConsumerConfig) SubjectBuilder[T]
}

func validateSubject(subject string) bool {
	if subject == "" {
		return false
	}
	for _, part := range strings.Split(subject, ".") {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			part = part[1 : len(part)-1]
		}
		if !nameRe.MatchString(part) {
			return false
		}
	}
	return true
}

type subjectOptions struct {
	subject               string
	args                  []string
	conn                  *nats.Conn
	stream                Stream
	consumerConfig        *jetstream.ConsumerConfig
	handlerProcessTimeout time.Duration
}

type subjectBuilder[T any] struct {
	subjectOptions
	jetStream                 jetstream.JetStream
	publisher                 Publisher[T]
	subscriber                Subscriber[T]
	streamPublisher           StreamPublisher[T]
	streamSubjectChecked      bool
	ephemeralConsumer         jetstream.Consumer
	durableConsumer           jetstream.Consumer
	ephemeralStreamSubscriber StreamSubscriber[T]
	durableStreamSubscriber   StreamSubscriber[T]
	consumerConfigChecked     bool
}

func (s *subjectBuilder[T]) getConn() *nats.Conn {
	if s.conn != nil {
		return s.conn
	}
	if s.stream != nil {
		return s.stream.Conn()
	}
	return getDefaultConn()
}

func (s *subjectBuilder[T]) getJetStream() jetstream.JetStream {
	if s.jetStream != nil {
		return s.jetStream
	}
	if s.stream != nil {
		return s.stream.JetStream()
	}
	conn := s.getConn()
	js, e := jetstream.New(conn)
	if e != nil {
		logrus.WithError(e).Panic("get jetstream failed")
	}
	s.jetStream = js
	return js
}

func (s *subjectBuilder[T]) getStream() (jetstream.Stream, errx.Error) {
	if s.stream == nil {
		return nil, errx.Newf("stream is not initialized for subject %s", s.subject)
	}
	st, err := s.stream.Stream()
	if err != nil {
		return nil, errx.Wrap(err).AppendMsgf("failed to get stream %s", s.subject).Err()
	}
	if s.streamSubjectChecked {
		return st, nil
	}
	if !isSubjectsContains(st.CachedInfo().Config.Subjects, s.subject) {
		return nil, errx.Newf("subject %s is not in stream %s", s.subject, st.CachedInfo().Config.Name)
	}
	s.streamSubjectChecked = true
	return st, nil
}

func (s *subjectBuilder[T]) WithArgs(args ...string) SubjectBuilder[T] {
	o := s.subjectOptions
	o.args = args
	return &subjectBuilder[T]{subjectOptions: o}
}

func (s *subjectBuilder[T]) WithConn(conn *nats.Conn) SubjectBuilder[T] {
	o := s.subjectOptions
	o.conn = conn
	return &subjectBuilder[T]{subjectOptions: o}
}

func (s *subjectBuilder[T]) WithStream(stream Stream) SubjectBuilder[T] {
	o := s.subjectOptions
	o.stream = stream
	sub, _ := replaceSubjectPlaceholders(o.subject, o.args...)
	if !isSubjectsContains(o.stream.Config().Subjects, sub) {
		logrus.Panicf("subject %s is not in stream %s", o.subject, o.stream.Config().Name)
	}
	return &subjectBuilder[T]{subjectOptions: o}
}

func (s *subjectBuilder[T]) WithConsumerConfig(consumerConfig jetstream.ConsumerConfig) SubjectBuilder[T] {
	o := s.subjectOptions
	o.consumerConfig = &consumerConfig
	return &subjectBuilder[T]{subjectOptions: o}
}

func (s *subjectBuilder[T]) WithHandlerProcessTimeout(timeout time.Duration) SubjectBuilder[T] {
	o := s.subjectOptions
	o.handlerProcessTimeout = timeout
	return &subjectBuilder[T]{subjectOptions: o}
}

func (s *subjectBuilder[T]) Conn() *nats.Conn {
	return s.getConn()
}

func (s *subjectBuilder[T]) JetStream() jetstream.JetStream {
	return s.getJetStream()
}

func (s *subjectBuilder[T]) Publisher() (Publisher[T], errx.Error) {
	if s.publisher != nil {
		return s.publisher, nil
	}
	res, err := newPublisher[T](s.getConn(), s.subject, s.args...)
	if err != nil {
		return nil, err
	}
	s.publisher = res
	return res, nil
}

func (s *subjectBuilder[T]) MustPublisher() Publisher[T] {
	res, err := s.Publisher()
	if err != nil {
		panic(err)
	}
	return res
}

func (s *subjectBuilder[T]) StreamPublisher() (StreamPublisher[T], errx.Error) {
	if s.streamPublisher != nil {
		return s.streamPublisher, nil
	}
	if _, err := s.getStream(); err != nil {
		return nil, err
	}
	res, err := newStreamPublisher[T](s.getJetStream(), s.subject, s.args...)
	if err != nil {
		return nil, errx.Wrap(err).AppendMsgf("new stream publisher failed for subject: %s", s.subject).Err()
	}
	s.streamPublisher = res
	return res, nil
}

func (s *subjectBuilder[T]) MustStreamPublisher() StreamPublisher[T] {
	res, err := s.StreamPublisher()
	if err != nil {
		panic(err)
	}
	return res
}

func (s *subjectBuilder[T]) Subscriber() Subscriber[T] {
	if s.subscriber != nil {
		return s.subscriber
	}
	sub, _ := replaceSubjectPlaceholders(s.subject, s.args...)
	s.subscriber = newSubscriber[T](s.getConn(), sub)
	return s.subscriber
}

func (s *subjectBuilder[T]) EphemeralConsumer() (jetstream.Consumer, errx.Error) {
	if s.ephemeralConsumer != nil {
		return s.ephemeralConsumer, nil
	}
	st, err := s.getStream()
	if err != nil {
		return nil, err
	}
	conf := defaultConsumerConfig
	if s.consumerConfig != nil {
		conf = *s.consumerConfig
		conf.Durable = ""
		conf.Name = ""
		if conf.Description == "" {
			conf.Description = fmt.Sprintf("create by subject builder for %s", s.subject)
		}
		if conf.FilterSubject == "" && len(conf.FilterSubjects) == 0 {
			conf.FilterSubject, _ = replaceSubjectPlaceholders(s.subject, s.args...)
		} else {
			if f := conf.FilterSubject; f != "" {
				sub, _ := replaceSubjectPlaceholders(s.subject, s.args...)
				if !isSubjectContains(sub, f) {
					return nil, errx.Newf("FilterSubject '%s' in consumer config does not contain filter subject: %s", f, sub)
				}
			} else if fs := conf.FilterSubjects; len(fs) > 0 {
				sub, _ := replaceSubjectPlaceholders(s.subject, s.args...)
				for _, f := range fs {
					if !isSubjectContains(sub, f) {
						return nil, errx.Newf("FilterSubject '%s' in consumer config does not contain filter subject: %s", f, sub)
					}
				}
			}
		}
	} else {
		conf.Description = fmt.Sprintf("create by subject builder for %s", s.subject)
		conf.FilterSubject, _ = replaceSubjectPlaceholders(s.subject, s.args...)
	}
	res, e := st.CreateConsumer(context.Background(), conf)
	if e != nil {
		return nil, errx.Wrap(e).Err()
	}
	s.ephemeralConsumer = res
	return res, nil
}

func (s *subjectBuilder[T]) DurableConsumer() (jetstream.Consumer, errx.Error) {
	if s.durableConsumer != nil {
		return s.durableConsumer, nil
	}
	st, err := s.getStream()
	if err != nil {
		return nil, err
	}
	conf := defaultConsumerConfig
	if s.consumerConfig != nil {
		conf = *s.consumerConfig
		if conf.Description == "" {
			conf.Description = fmt.Sprintf("create by subject builder for %s", s.subject)
		}
		if conf.FilterSubject == "" && len(conf.FilterSubjects) == 0 {
			conf.FilterSubject, _ = replaceSubjectPlaceholders(s.subject, s.args...)
		} else {
			if f := conf.FilterSubject; f != "" {
				sub, _ := replaceSubjectPlaceholders(s.subject, s.args...)
				if !isSubjectContains(sub, f) {
					return nil, errx.Newf("FilterSubject '%s' in consumer config does not contain filter subject: %s", f, sub)
				}
			} else if fs := conf.FilterSubjects; len(fs) > 0 {
				sub, _ := replaceSubjectPlaceholders(s.subject, s.args...)
				for _, f := range fs {
					if !isSubjectContains(sub, f) {
						return nil, errx.Newf("FilterSubject '%s' in consumer config does not contain filter subject: %s", f, sub)
					}
				}
			}
		}
	} else {
		conf.Description = fmt.Sprintf("create by subject builder for %s", s.subject)
		conf.FilterSubject, _ = replaceSubjectPlaceholders(s.subject, s.args...)
	}
	if conf.Durable == "" {
		durable, _ := replaceSubjectPlaceholders(s.subject, s.args...)
		durable = strings.ReplaceAll(durable, "*", "all")
		durable = strings.ReplaceAll(durable, ".", "_")
		conf.Durable = durable
	}
	ctx := context.Background()
	res, e := st.Consumer(ctx, conf.Durable)
	if e != nil {
		if !errors.Is(e, jetstream.ErrConsumerNotFound) {
			return nil, errx.Wrap(e).AppendMsgf("failed to get consumer %s", conf.Durable).Err()
		}
		res, e = st.CreateOrUpdateConsumer(ctx, conf)
		if e != nil {
			return nil, errx.Wrap(e).AppendMsgf("failed to create consumer %s", conf.Durable).Err()
		}
		logrus.Infof("consumer %s created", conf.Durable)
		s.durableConsumer = res
		return res, nil
	}
	if !compareConsumerConfig(res.CachedInfo().Config, conf) {
		res, e = st.UpdateConsumer(ctx, conf)
		if e != nil {
			return nil, errx.Wrap(e).AppendMsgf("failed to update consumer %s", conf.Durable).Err()
		}
		logrus.Infof("consumer %s updated", conf.Durable)
	}
	s.durableConsumer = res
	return res, nil
}

func (s *subjectBuilder[T]) EphemeralStreamSubscriber() (StreamSubscriber[T], errx.Error) {
	if s.ephemeralStreamSubscriber != nil {
		return s.ephemeralStreamSubscriber, nil
	}
	c, err := s.EphemeralConsumer()
	if err != nil {
		return nil, err
	}
	s.ephemeralStreamSubscriber = newStreamSubscriber[T](c, s.handlerProcessTimeout)
	return s.ephemeralStreamSubscriber, nil
}

func (s *subjectBuilder[T]) MustEphemeralStreamSubscriber() StreamSubscriber[T] {
	res, err := s.EphemeralStreamSubscriber()
	if err != nil {
		panic(err)
	}
	return res
}

func (s *subjectBuilder[T]) DurableStreamSubscriber() (StreamSubscriber[T], errx.Error) {
	if s.durableStreamSubscriber != nil {
		return s.durableStreamSubscriber, nil
	}
	c, err := s.EphemeralConsumer()
	if err != nil {
		return nil, err
	}
	s.durableStreamSubscriber = newStreamSubscriber[T](c, s.handlerProcessTimeout)
	return s.durableStreamSubscriber, nil
}

func (s *subjectBuilder[T]) MustDurableStreamSubscriber() StreamSubscriber[T] {
	res, err := s.DurableStreamSubscriber()
	if err != nil {
		panic(err)
	}
	return res
}

func compareConsumerConfig(c1, c2 jetstream.ConsumerConfig) bool {
	if c1.Name != c2.Name {
		return false
	}
	if c1.Durable != c2.Durable {
		return false
	}
	if c1.Description != c2.Description {
		return false
	}
	if c1.DeliverPolicy != c2.DeliverPolicy {
		return false
	}
	if c1.OptStartSeq != c2.OptStartSeq {
		return false
	}
	if !reflect.DeepEqual(c1.OptStartTime, c2.OptStartTime) {
		return false
	}
	if c1.AckPolicy != c2.AckPolicy {
		return false
	}
	if c1.AckWait != c2.AckWait {
		return false
	}
	if c1.MaxDeliver != c2.MaxDeliver {
		return false
	}
	if !reflect.DeepEqual(c1.BackOff, c2.BackOff) {
		return false
	}
	if c1.FilterSubject != c2.FilterSubject {
		return false
	}
	if c1.ReplayPolicy != c2.ReplayPolicy {
		return false
	}
	if c1.RateLimit != c2.RateLimit {
		return false
	}
	if c1.SampleFrequency != c2.SampleFrequency {
		return false
	}
	if c1.MaxWaiting != c2.MaxWaiting {
		return false
	}
	if c1.MaxAckPending != c2.MaxAckPending {
		return false
	}
	if c1.HeadersOnly != c2.HeadersOnly {
		return false
	}
	if c1.MaxRequestBatch != c2.MaxRequestBatch {
		return false
	}
	if c1.MaxRequestExpires != c2.MaxRequestExpires {
		return false
	}
	if c1.MaxRequestMaxBytes != c2.MaxRequestMaxBytes {
		return false
	}
	if c1.InactiveThreshold != c2.InactiveThreshold {
		return false
	}
	if c1.Replicas != c2.Replicas {
		return false
	}
	if c1.MemoryStorage != c2.MemoryStorage {
		return false
	}
	if !reflect.DeepEqual(c1.FilterSubjects, c2.FilterSubjects) {
		return false
	}
	if !reflect.DeepEqual(c1.Metadata, c2.Metadata) {
		return false
	}
	return true
}

var defaultConsumerConfig = jetstream.ConsumerConfig{
	DeliverPolicy: jetstream.DeliverLastPolicy,
}
