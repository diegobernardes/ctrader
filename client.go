package ctrader

import (
	"context"
	"errors"
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

func (c *Client) SendRequest(ctx context.Context, req proto.Message) (proto.Message, error) {
	return c.send(ctx, req, true)
}

func (c *Client) SendEvent(ctx context.Context, req proto.Message) error {
	_, err := c.send(ctx, req, false)
	return err
}

func (c *Client) AccountAuth(
	ctx context.Context, accountIDRaw int, token string,
) (*openapi.ProtoOAAccountAuthRes, error) {
	accountID := int64(accountIDRaw)
	req := &openapi.ProtoOAAccountAuthReq{
		AccessToken:         &token,
		CtidTraderAccountId: &accountID,
	}
	resp, err := c.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute the request: %w", err)
	}
	switch v := resp.(type) {
	case *openapi.ProtoOAErrorRes:
		return nil, errors.New("failed authenticate the account")
	case *openapi.ProtoOAAccountAuthRes:
		return v, nil
	default:
		return nil, errors.New("unexpected response type")
	}
}

func (c *Client) SymbolList(ctx context.Context, accountIDRaw int) (*openapi.ProtoOASymbolsListRes, error) {
	accountID := int64(accountIDRaw)
	req := &openapi.ProtoOASymbolsListReq{
		CtidTraderAccountId: &accountID,
	}
	resp, err := c.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute the request: %w", err)
	}
	switch v := resp.(type) {
	case *openapi.ProtoOAErrorRes:
		return nil, errors.New("failed to fetch the symbol list")
	case *openapi.ProtoOASymbolsListRes:
		return v, nil
	default:
		return nil, errors.New("unexpected response type")
	}
}

func (c *Client) Subscribe(
	ctx context.Context, accountIDRaw int, symbols []int64,
) (*openapi.ProtoOASubscribeSpotsRes, error) {
	accountID := int64(accountIDRaw)
	req := &openapi.ProtoOASubscribeSpotsReq{
		CtidTraderAccountId: &accountID,
		SymbolId:            symbols,
	}
	resp, err := c.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute the request: %w", err)
	}
	switch v := resp.(type) {
	case *openapi.ProtoOAErrorRes:
		return nil, errors.New("failed to subscribe")
	case *openapi.ProtoOASubscribeSpotsRes:
		return v, nil
	default:
		return nil, errors.New("unexpected response type")
	}
}

func (c *Client) Version(ctx context.Context) (*openapi.ProtoOAVersionRes, error) {
	req := &openapi.ProtoOAVersionReq{}
	resp, err := c.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute the request: %w", err)
	}
	switch v := resp.(type) {
	case *openapi.ProtoOAErrorRes:
		return nil, errors.New("failed to fetch the version")
	case *openapi.ProtoOAVersionRes:
		return v, nil
	default:
		return nil, errors.New("unexpected response type")
	}
}

func (c *Client) applicationAuthorization(ctx context.Context) error {
	req := &openapi.ProtoOAApplicationAuthReq{
		ClientId:     &c.ApplicationClientID,
		ClientSecret: &c.ApplicationSecret,
	}
	resp, err := c.SendRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send the message: %w", err)
	}
	switch resp.(type) {
	case *openapi.ProtoOAErrorRes:
		return errors.New("failed to authorize an application")
	case *openapi.ProtoOAApplicationAuthRes:
		return nil
	default:
		return errors.New("unexpected response type")
	}
}

func (c *Client) handlerMessage(payload []byte) {
	var msg openapi.ProtoMessage
	if err := proto.Unmarshal(payload, &msg); err != nil {
		c.Logger.Error("failed to unmarshal message", "error", err)
		return
	}
	if msg.ClientMsgId == nil {
		message, err := mappingResponse(*msg.PayloadType)
		if err != nil {
			c.Logger.Error("unknow message type", "error", err)
			return
		}
		if err = proto.Unmarshal(msg.Payload, message); err != nil {
			c.Logger.Error("failed to unmarshal payload", "error", err)
			return
		}
		c.HandlerEvent(message)
	} else {
		c.requestRegistryMutex.Lock()
		chanResponse, ok := c.requestRegistry[*msg.ClientMsgId]
		c.requestRegistryMutex.Unlock()
		if !ok {
			c.Logger.Error("client message ID not found", "clientMessageID", *msg.ClientMsgId)
			return
		}
		chanResponse <- &msg
	}
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
			if err := c.SendEvent(context.Background(), &req); err != nil {
				c.handlerError(fmt.Errorf("failed to send the heartbeat event: %w", err))
			}
		}
	}()
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

func (c *Client) send(ctx context.Context, req proto.Message, isRequest bool) (proto.Message, error) {
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

	var chanResponse chan *openapi.ProtoMessage
	if isRequest {
		chanResponse = make(chan *openapi.ProtoMessage, 1)
		c.requestRegistryMutex.Lock()
		c.requestRegistry[id] = chanResponse
		c.requestRegistryMutex.Unlock()
		defer delete(c.requestRegistry, id)
	}

	if errSend := c.transport.send(payload); errSend != nil {
		return nil, fmt.Errorf("failed to send the message: %w", errSend)
	}

	if !isRequest {
		return nil, nil
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	case messageBase := <-chanResponse:
		message, errMessage := mappingResponse(*messageBase.PayloadType)
		if errMessage != nil {
			return nil, fmt.Errorf("failed to get the response type: %w", errMessage)
		}
		if err = proto.Unmarshal(messageBase.Payload, message); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the response: %w", err)
		}
		return message, nil
	}
}
