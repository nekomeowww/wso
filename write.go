package wso

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nekomeowww/fo"
)

// writePump 向连接中写入消息
//
// NOTICE: 该函数将会默认每间隔 30 秒向连接中写入一个 Ping 消息，用于探测死亡的客户端连接
func (w *Websocket) writePump() {
	ticker := time.NewTicker(w.options.pingMessagePeriod)
	defer func() {
		ticker.Stop()
		w.Close()
	}()
	for {
		select {
		case <-w.closeChannel:
			return
		case message, ok := <-w.writeChannel:
			ctx, cancel := context.WithTimeout(context.Background(), w.options.writeDeadline)
			defer cancel()

			if !ok {
				_ = fo.Invoke0(ctx, func() error {
					err := w.websocketConn.WriteMessage(websocket.CloseMessage, []byte{})
					w.options.logger.Error(err)

					return nil
				})

				return
			}

			err := fo.Invoke0(ctx, func() error { return w.websocketConn.WriteMessage(websocket.TextMessage, message) })
			if err != nil {
				w.options.logger.Warnf("websocket: failed to write message: %v", err)
				return
			}
		case <-ticker.C:
			// 每次发送 Ping 消息都会重置写入超时时间
			_ = w.websocketConn.SetWriteDeadline(time.Now().Add(w.options.writeDeadline))

			{
				ctx, cancel := context.WithTimeout(context.Background(), w.options.writeDeadline)
				defer cancel()

				// 发送 Ping 消息
				err := fo.Invoke0(ctx, func() error { return w.Ping() })
				if err != nil {
					w.options.logger.Warnf("websocket: failed to send ping message: %v", err)
					return
				}
			}
		}
	}
}
