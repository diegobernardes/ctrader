package ctrader

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diegobernardes/ctrader/openapi"
)

func (c *Client[T]) AccountAuth(
	ctx context.Context, accountIDRaw int, token string,
) (*openapi.ProtoOAAccountAuthRes, error) {
	accountID := int64(accountIDRaw)
	req := &openapi.ProtoOAAccountAuthReq{
		AccessToken:         &token,
		CtidTraderAccountId: &accountID,
	}
	resp, err := c.send(context.Background(), req, int32(openapi.ProtoOAPayloadType_PROTO_OA_ACCOUNT_AUTH_REQ), true)
	if err != nil {
		return nil, fmt.Errorf("failed to execute the request: %w", err)
	}
	switch v := resp.(type) {
	case *openapi.ProtoOAErrorRes:
		return nil, errors.New("failed to fetch the symbol list")
	case *openapi.ProtoOAAccountAuthRes:
		return v, nil
	default:
		return nil, errors.New("unexpected response type")
	}
}

func (c *Client[T]) SymbolList(ctx context.Context, accountIDRaw int) (*openapi.ProtoOASymbolsListRes, error) {
	accountID := int64(accountIDRaw)
	req := &openapi.ProtoOASymbolsListReq{
		CtidTraderAccountId: &accountID,
	}
	resp, err := c.send(ctx, req, int32(openapi.ProtoOAPayloadType_PROTO_OA_SYMBOLS_LIST_REQ), true)
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

// ProtoOaSpotEvent
func (c *Client[T]) Subscribe(
	ctx context.Context, accountIDRaw int, symbols []int64,
) (*openapi.ProtoOASubscribeSpotsRes, error) {
	accountID := int64(accountIDRaw)
	req := &openapi.ProtoOASubscribeSpotsReq{
		CtidTraderAccountId: &accountID,
		SymbolId:            symbols,
	}
	resp, err := c.send(ctx, req, int32(openapi.ProtoOAPayloadType_PROTO_OA_SUBSCRIBE_SPOTS_REQ), true)
	if err != nil {
		return nil, fmt.Errorf("failed to execute the request: %w", err)
	}
	switch v := resp.(type) {
	case *openapi.ProtoOAErrorRes:
		return nil, errors.New("failed to fetch the symbol list")
	case *openapi.ProtoOASubscribeSpotsRes:
		return v, nil
	default:
		return nil, errors.New("unexpected response type")
	}
}

func (c *Client[T]) applicationAuthorization(ctx context.Context) error {
	req := &openapi.ProtoOAApplicationAuthReq{
		ClientId:     &c.ClientID,
		ClientSecret: &c.Secret,
	}
	resp, err := c.send(ctx, req, int32(openapi.ProtoOAPayloadType_PROTO_OA_APPLICATION_AUTH_REQ), true)
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

func (c *Client[T]) keepalive() {
	c.wg.Add(1)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer func() {
			ticker.Stop()
			c.wg.Done()
		}()
		for range ticker.C {
			if c.stopSignal.Load() {
				return
			}
			payloadTypeRaw := openapi.ProtoPayloadType_HEARTBEAT_EVENT
			payloadType := uint32(payloadTypeRaw)
			req := openapi.ProtoMessage{
				PayloadType: &payloadType,
			}
			if _, err := c.send(context.Background(), &req, int32(payloadTypeRaw), false); err != nil {
				c.handlerError(fmt.Errorf("failed to send the heartbeat event: %w", err))
			}
		}
	}()
}
