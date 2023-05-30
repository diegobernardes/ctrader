package ctrader

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/diegobernardes/ctrader/openapi"
)

func TestClientKeepAlive(t *testing.T) {
	t.Parallel()
	mc := mockClient{t: t}
	c := Client[*mockClient]{Transport: &mc}
	c.keepalive()
	time.Sleep(21 * time.Second)
	require.Equal(t, mc.count.Load(), int64(2))
	c.stopSignal.Store(true)
	time.Sleep(11 * time.Second)
	require.Equal(t, mc.count.Load(), int64(2))
}

type mockClient struct {
	mock.Mock
	clientTransport
	t     *testing.T
	count atomic.Int64
}

func (m *mockClient) send(payload []byte) error {
	var msg openapi.ProtoMessage
	require.NoError(m.t, proto.Unmarshal(payload, &msg))
	require.Equal(m.t, *msg.PayloadType, uint32(openapi.ProtoPayloadType_HEARTBEAT_EVENT))
	m.count.Add(1)
	return nil
}
