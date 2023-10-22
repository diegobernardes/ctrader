package ctrader

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/satori/uuid"
	"golang.org/x/exp/slog"
	"google.golang.org/protobuf/proto"

	"github.com/diegobernardes/ctrader/openapi"
)

type clientTransport interface {
	start(string) error
	stop() error
	send([]byte) error
	setHandler(func([]byte), func(error))
}

type Client struct {
	ApplicationClientID string
	ApplicationSecret   string
	HandlerEvent        func(proto.Message)
	Deadline            time.Duration
	Logger              *slog.Logger
	Live                bool

	transport            clientTransport
	stopSignal           atomic.Bool
	wg                   sync.WaitGroup
	requestRegistry      map[string]chan *openapi.ProtoMessage
	requestRegistryMutex sync.Mutex
}

func (c *Client) Start() error {
	c.transport = &transportTCP{deadline: c.Deadline}
	var address string
	if c.Live {
		address = "live.ctraderapi.com:5035"
	} else {
		address = "demo.ctraderapi.com:5035"
	}
	c.transport.setHandler(c.handlerMessage, c.handlerError)
	if err := c.transport.start(address); err != nil {
		return fmt.Errorf("failed to open the transport: %w", err)
	}
	c.requestRegistry = make(map[string]chan *openapi.ProtoMessage)
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second)
	defer ctxCancel()
	if err := c.applicationAuthorization(ctx); err != nil {
		return fmt.Errorf("failed to authenticate the application: %w", err)
	}
	c.keepalive()
	return nil
}

func (c *Client) Stop() error {
	c.stopSignal.Store(true)
	c.wg.Wait()
	if err := c.transport.stop(); err != nil {
		return fmt.Errorf("failed to close the transport: %w", err)
	}
	return nil
}

func (c *Client) handlerMessage(payload []byte) {
	var msg openapi.ProtoMessage
	if err := proto.Unmarshal(payload, &msg); err != nil {
		c.Logger.Error("failed to unmarshal message", "error", err)
		return
	}
	if msg.GetClientMsgId() == "" {
		message, err := mappingResponse(msg.GetPayloadType())
		if err != nil {
			c.Logger.Error("unknow message type", "error", err)
			return
		}
		if err = proto.Unmarshal(msg.GetPayload(), message); err != nil {
			c.Logger.Error("failed to unmarshal payload", "error", err)
			return
		}
		c.HandlerEvent(message)
	} else {
		c.requestRegistryMutex.Lock()
		chanResponse, ok := c.requestRegistry[msg.GetClientMsgId()]
		c.requestRegistryMutex.Unlock()
		if !ok {
			c.Logger.Error("client message ID not found", "clientMessageID", msg.GetClientMsgId())
			return
		}
		chanResponse <- &msg
	}
}

func (c *Client) handlerError(err error) {
	c.Logger.Error("Asynchronous error", "error", err.Error())
	for {
		if err = c.Stop(); err != nil {
			c.Logger.Error("failed to stop the client", "error", err.Error())
			time.Sleep(time.Second)
			continue
		}
		if err = c.Start(); err != nil {
			c.Logger.Error("failed to start the client", "error", err.Error())
			time.Sleep(time.Second)
			continue
		}
		break
	}
}

func (c *Client) sendRequest(ctx context.Context, req proto.Message) (proto.Message, error) {
	payloadType, err := mappingPayloadType(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get the payload type: %w", err)
	}

	payloadBase, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal base request: %w", err)
	}

	id := uuid.NewV4().String()
	reqType := uint32(payloadType)
	message := openapi.ProtoMessage{
		ClientMsgId: &id,
		Payload:     payloadBase,
		PayloadType: &reqType,
	}
	payload, err := proto.Marshal(&message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	chanResponse := make(chan *openapi.ProtoMessage, 1)
	c.requestRegistryMutex.Lock()
	c.requestRegistry[id] = chanResponse
	c.requestRegistryMutex.Unlock()
	defer delete(c.requestRegistry, id)

	if errSend := c.transport.send(payload); errSend != nil {
		return nil, fmt.Errorf("failed to send the message: %w", errSend)
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	case messageBase := <-chanResponse:
		message, errMessage := mappingResponse(messageBase.GetPayloadType())
		if errMessage != nil {
			return nil, fmt.Errorf("failed to get the response type: %w", errMessage)
		}
		if err = proto.Unmarshal(messageBase.GetPayload(), message); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the response: %w", err)
		}
		return message, nil
	}
}

func (c *Client) sendEvent(ctx context.Context, e proto.Message) error {
	payload, err := proto.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}
	if errSend := c.transport.send(payload); errSend != nil {
		return fmt.Errorf("failed to send the message: %w", errSend)
	}
	return nil
}

func (c *Client) applicationAuthorization(ctx context.Context) error {
	req := &openapi.ProtoOAApplicationAuthReq{
		ClientId:     &c.ApplicationClientID,
		ClientSecret: &c.ApplicationSecret,
	}
	_, err := Command[*openapi.ProtoOAApplicationAuthReq, *openapi.ProtoOAApplicationAuthRes](ctx, c, req)
	if err != nil {
		return fmt.Errorf("failed to send the message: %w", err)
	}
	return nil
}

func (c *Client) keepalive() {
	c.wg.Add(1)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer func() {
			ticker.Stop()
			c.wg.Done()
		}()
		payloadTypeRaw := openapi.ProtoPayloadType_HEARTBEAT_EVENT
		payloadType := uint32(payloadTypeRaw)
		req := openapi.ProtoMessage{
			PayloadType: &payloadType,
		}
		for range ticker.C {
			if c.stopSignal.Load() {
				return
			}
			if err := c.sendEvent(context.Background(), &req); err != nil {
				c.handlerError(fmt.Errorf("failed to send the heartbeat event: %w", err))
			}
		}
	}()
}
