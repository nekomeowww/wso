package wso

import (
	"time"

	"github.com/gorilla/websocket"
)

// readPump 读取连接中的消息，当收到消息时将会调用所有注册的消息监听器
func (w *Websocket) readPump() {
	// 每次新建连接都配置一次 Read 消息的超时时间
	_ = w.websocketConn.SetReadDeadline(time.Now().Add(w.options.readDeadline))

	// defer 语句将会在函数退出时调用，用于关闭连接
	defer func() {
		w.Close()
	}()
	for {
		// 读取消息
		messageType, messageBody, err := w.websocketConn.ReadMessage()
		if err != nil {
			// 如果不是因为 websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure 而关闭的连接，则认为是错误
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// 记录日志
				w.options.logger.Warnf("websocket: failed to read message: %v", err)
			}

			// 退出函数，调用 defer 语句
			return
		}

		// 每次成功收到消息之后都配置一次 Read 消息的超时时间
		// 根据 [Documentation is not enough clear about ping/pong](https://github.com/gorilla/websocket/issues/649) Issue 所讨论的，应当使用 SetReadDeadline 来探测死亡的客户端连接
		_ = w.websocketConn.SetReadDeadline(time.Now().Add(w.options.readDeadline))

		// 发送消息到所有关联的消息监听器
		if w.options.processor != nil {
			var processingErr error

			switch messageType {
			case websocket.TextMessage:
				processingErr = w.options.processor.OnTextMessage(w, messageBody)
			case websocket.BinaryMessage:
				processingErr = w.options.processor.OnBinaryMessage(w, messageBody)
			}

			if processingErr != nil {
				w.options.logger.Warnf("websocket: failed to process message: %v", processingErr)
				w.CloseOnError(err)
			}
		}
	}
}
