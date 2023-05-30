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
	t.Run("TransportTCP", testClientIntegrationRunner(&TransportTCP{Deadline: time.Second}))
}

func testClientIntegrationRunner[T clientTransport](transport T) func(*testing.T) {
	return func(t *testing.T) {
		c := &Client[T]{
			Transport: transport,
			Logger:    slog.New(slog.NewTextHandler(os.Stdout, nil)),
			ClientID:  os.Getenv("CTRADER_CLIENT_ID"),
			Secret:    os.Getenv("CTRADER_SECRET"),
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
}
