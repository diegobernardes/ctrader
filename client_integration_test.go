//go:build integration

package ctrader

import (
	"context"
	"os"
	"strconv"
	"sync"
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

	acc, err := c.AccountAuth(context.Background(), ctraderAccountID, ctraderToken)
	require.NoError(t, err)
	require.Equal(t, ctraderAccountID, int(*acc.CtidTraderAccountId))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, err := c.SymbolList(context.Background(), ctraderAccountID)
		require.NoError(t, err)
		_, ok := lo.Find(resp.Symbol, func(s *openapi.ProtoOALightSymbol) bool {
			return *s.SymbolName == "EURUSD"
		})
		require.True(t, ok)
	}()
	wg.Wait()
	require.NoError(t, c.Stop())
}
