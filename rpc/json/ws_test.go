package json

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/rpc/v2/json2"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

type NestedRPCResponse struct {
	Query string `json:"query"`
	Data  struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"data"`
}

func TestWebSockets(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	_, local := getRPC(t)
	handler, err := GetHTTPHandler(local, log.TestingLogger())
	require.NoError(err)

	srv := httptest.NewServer(handler)

	conn, resp, err := websocket.DefaultDialer.Dial(strings.Replace(srv.URL, "http://", "ws://", 1)+"/websocket", nil)
	require.NoError(err)
	require.NotNil(resp)
	require.NotNil(conn)
	defer func() {
		_ = conn.Close()
	}()

	assert.Equal(http.StatusSwitchingProtocols, resp.StatusCode)

	err = conn.WriteMessage(websocket.TextMessage, []byte(`
{
    "jsonrpc": "2.0",
    "method": "subscribe",
    "id": 7,
    "params": {
        "query": "tm.event='NewBlock'"
    }
}
`))
	assert.NoError(err)

	err = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	assert.NoError(err)
	typ, msg, err := conn.ReadMessage()
	assert.NoError(err)
	assert.Equal(websocket.TextMessage, typ)
	assert.NotEmpty(msg)

	// wait for new block event
	err = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	assert.NoError(err)
	typ, msg, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal(websocket.TextMessage, typ)
	assert.NotEmpty(msg)
	var responsePayload rpctypes.RPCResponse
	err = json.Unmarshal(msg, &responsePayload)
	assert.NoError(err)
	assert.Equal(rpctypes.JSONRPCIntID(7), responsePayload.ID)
	var m map[string]interface{}
	err = json.Unmarshal([]byte(responsePayload.Result), &m)
	require.NoError(err)

	// TODO(omritoptix): json unmarshalling of the dataPayload fails as dataPayload was encoded with amino and not json (and as such encodes 64bit numbers as strings).
	// we need to unmarshal using the tendermint json library for it to populate the dataPayload correctly. Currently skipping this part of the test.
	// valueField := m["data"].(map[string]interface{})["value"]
	// valueJSON, err := json.Marshal(valueField)
	// var dataPayload tmtypes.EventDataNewBlock
	// err = tmjson.Unmarshal(valueJSON, &dataPayload)
	// require.NoError(err)
	// assert.NotNil(dataPayload.ResultBeginBlock)
	// assert.NotNil(dataPayload.Block)
	// assert.GreaterOrEqual(dataPayload.Block.Height, int64(1))
	// assert.NotNil(dataPayload.ResultEndBlock)

	unsubscribeAllReq, err := json2.EncodeClientRequest("unsubscribe_all", &unsubscribeAllArgs{})
	require.NoError(err)
	require.NotEmpty(unsubscribeAllReq)
	req := httptest.NewRequest(http.MethodGet, "/", bytes.NewReader(unsubscribeAllReq))
	req.RemoteAddr = conn.LocalAddr().String()
	rsp := httptest.NewRecorder()
	handler.ServeHTTP(rsp, req)
	assert.Equal(http.StatusOK, rsp.Code)
	jsonResp := response{}
	assert.NoError(json.Unmarshal(rsp.Body.Bytes(), &jsonResp))
	assert.Nil(jsonResp.Error)
}

// Test that when a websocket connection is closed, the corresponding
// subscription is also removed.
func TestWebsocketCloseUnsubscribe(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	_, local := getRPC(t)
	handler, err := GetHTTPHandler(local, log.TestingLogger())
	require.NoError(err)

	srv := httptest.NewServer(handler)

	conn, resp, err := websocket.DefaultDialer.Dial(strings.Replace(srv.URL, "http://", "ws://", 1)+"/websocket", nil)
	require.NoError(err)
	require.NotNil(resp)
	require.NotNil(conn)
	assert.Equal(http.StatusSwitchingProtocols, resp.StatusCode)
	subscribed_clients := local.EventBus.NumClients()

	err = conn.WriteMessage(websocket.TextMessage, []byte(`
{
    "jsonrpc": "2.0",
    "method": "subscribe",
    "id": 7,
    "params": {
        "query": "tm.event='NewBlock'"
    }
}
`))
	assert.NoError(err)
	// Vaildate we have a new client
	assert.Eventually(func() bool {
		return subscribed_clients+1 == local.EventBus.NumClients()
	}, 3*time.Second, 100*time.Millisecond)
	// disconnect websocket
	err = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	assert.NoError(err)
	// Validate we have one less client
	assert.Eventually(func() bool {
		return subscribed_clients == local.EventBus.NumClients()
	}, 3*time.Second, 100*time.Millisecond)

}
