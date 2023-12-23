//go:build integration

package ctrader

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"

	"github.com/diegobernardes/ctrader/openapi"
)

func TestClientIntegration(t *testing.T) {
	c := &Client{
		Logger:              slog.New(slog.NewTextHandler(os.Stdout, nil)),
		ApplicationClientID: os.Getenv("CTRADER_CLIENT_ID"),
		ApplicationSecret:   os.Getenv("CTRADER_SECRET"),
		Deadline:            time.Second,
	}

	ctraderAccountIDRaw := os.Getenv("CTRADER_ACCOUNT_ID")
	ctraderAccountID, err := strconv.Atoi(ctraderAccountIDRaw)
	require.NoError(t, err)
	ctraderToken := os.Getenv("CTRADER_TOKEN")

	require.NoError(t, c.Start())

	req := &openapi.ProtoOAAccountAuthReq{
		AccessToken:         &ctraderToken,
		CtidTraderAccountId: lo.ToPtr(int64(ctraderAccountID)),
	}
	r2, err := Command[*openapi.ProtoOAAccountAuthReq, *openapi.ProtoOAAccountAuthRes](context.Background(), c, req)
	require.NoError(t, err)
	require.Equal(t, ctraderAccountID, int(*r2.CtidTraderAccountId))

	reqSymbolList := &openapi.ProtoOASymbolsListReq{
		CtidTraderAccountId: lo.ToPtr(int64(ctraderAccountID)),
	}
	respSymbolList, errSymbolList := Command[*openapi.ProtoOASymbolsListReq, *openapi.ProtoOASymbolsListRes](
		context.Background(), c, reqSymbolList,
	)
	require.NoError(t, errSymbolList)
	_, ok := lo.Find(respSymbolList.Symbol, func(s *openapi.ProtoOALightSymbol) bool {
		return *s.SymbolName == "EURUSD"
	})
	require.True(t, ok)
	require.NoError(t, c.Stop())
}
