package wso

import (
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestWebsocket(t *testing.T) {
	recorder := httptest.NewRecorder()
	gin.SetMode("release")
	c, _ := gin.CreateTestContext(recorder)

	// 在测试上下文中添加 Param ID
	c.Params = append(c.Params, gin.Param{Key: "param1", Value: "abcd"})
	c.Params = append(c.Params, gin.Param{Key: "param2", Value: "1"})

	clientWebsocketConn, serverWebsocket, cleanupWebsocket := NewTestGinWebsocket(t, c)
	defer cleanupWebsocket()

	serverWebsocket.Open()

	err := clientWebsocketConn.WriteMessage(websocket.TextMessage, []byte(`{"type": "sync"}`))
	require.NoError(t, err)

	var clientMessage string
	var waitGroup sync.WaitGroup

	waitGroup.Add(1)
	go func() {
		for {
			_, messageBody, err := clientWebsocketConn.ReadMessage()
			require.NoError(t, err)

			clientMessage = string(messageBody)
			break
		}

		waitGroup.Done()
	}()

	serverWebsocket.WriteJSON(map[string]interface{}{
		"type": "sync",
	})

	waitGroup.Wait()
	assert.Equal(t, `{"type":"sync"}`, clientMessage)
}
