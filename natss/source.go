package natss

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/go-nats-streaming/pb"
	"github.com/utilitywarehouse/go-pubsub"
)

func generateID() string {
	random := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	_, err := rand.Read(random)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(random)
}

var _ pubsub.MessageSource = (*messageSource)(nil)

type MessageSourceConfig struct {
	NatsURL    string
	ClusterID  string
	Topic      string
	ConsumerID string
}

type messageSource struct {
	natsURL    string
	clusterID  string
	consumerID string
	topic      string
}

func NewMessageSource(conf MessageSourceConfig) (pubsub.MessageSource, error) {
	return &messageSource{
		natsURL:    conf.NatsURL,
		clusterID:  conf.ClusterID,
		consumerID: conf.ConsumerID,
		topic:      conf.Topic,
	}, nil
}

func (mq *messageSource) ConsumeMessages(ctx context.Context, handler pubsub.ConsumerMessageHandler, onError pubsub.ConsumerErrorHandler) error {

	conn, err := stan.Connect(mq.clusterID, mq.consumerID+generateID(), stan.NatsURL(mq.natsURL))
	if err != nil {
		return err
	}
	defer conn.Close()

	consumeErrs := make(chan error, 1)

	broken := false

	f := func(msg *stan.Msg) {

		if broken {
			return
		}

		m := pubsub.ConsumerMessage{msg.Data}
		err := handler(m)
		if err != nil {
			if err := onError(m, err); err != nil {
				broken = true
				consumeErrs <- err
			} else {
				msg.Ack()
			}
		} else {
			msg.Ack()
		}
	}

	startOpt := stan.StartAt(pb.StartPosition_First)

	subcription, err := conn.QueueSubscribe(mq.topic, mq.consumerID, f, startOpt, stan.DurableName(mq.consumerID), stan.SetManualAckMode())

	defer subcription.Close()

	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
	case err = <-consumeErrs:
	}

	//conn.Close()

	return err
}

func (mq *messageSource) Status() (*pubsub.Status, error) {
	return nil, errors.New("status is not implemented")
}
