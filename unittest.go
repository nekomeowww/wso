package wso

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// NewTestWebsocket 新建一个 Websocket 测试客户端
func NewTestGinWebsocket(t *testing.T, c *gin.Context) (*websocket.Conn, *Websocket, func()) {
	var serverWebsocket *Websocket
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		websocketConn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		conn := NewWebsocketFromConn(websocketConn, c.Writer, c.Request, WithOnMessageSkipStrategy(true, true))

		serverWebsocket = conn
	}

	websocketServer := httptest.NewServer(http.HandlerFunc(handlerFunc))
	clientWebsocketConn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(websocketServer.URL, "http"), nil)
	require.NoError(t, err)

	err = clientWebsocketConn.WriteMessage(websocket.TextMessage, []byte("❤️"))
	require.NoError(t, err)

	return clientWebsocketConn, serverWebsocket, func() {
		clientWebsocketConn.Close()
		websocketServer.Close()
	}
}
