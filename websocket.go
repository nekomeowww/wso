package wso

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Websocket 封装了 Websocket 连接及其交互方式
type Websocket struct {
	requestURI string
	remoteAddr string

	alreadyClosed bool

	websocketConn *websocket.Conn
	writeChannel  chan []byte
	closeChannel  chan struct{}
	options       *websocketOptions
}

// NewWebsocket 创建新的 Websocket 连接封装，需要传递 websocket.Upgrader 用于升级 HTTP 连接，connectionPool 用于管理连接，context 用于操作和传递请求上下文
func NewWebsocket(upgrader websocket.Upgrader, w http.ResponseWriter, r *http.Request, callOptions ...WebsocketCallOption) (*Websocket, error) {
	// 升级 HTTP 连接
	websocketConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	return NewWebsocketFromConn(websocketConn, w, r, callOptions...), nil
}

// NewWebsocketFromConn 创建新的 Websocket 连接封装，需要传递 websocket.Conn 用于读写消息，connectionPool 用于管理连接，context 用于操作和传递请求上下文
func NewWebsocketFromConn(websocketConn *websocket.Conn, w http.ResponseWriter, r *http.Request, callOptions ...WebsocketCallOption) *Websocket {
	// 创建 Websocket 连接封装
	ws := &Websocket{
		requestURI:    r.RequestURI,
		remoteAddr:    r.RemoteAddr,
		websocketConn: websocketConn,
		writeChannel:  make(chan []byte),
		closeChannel:  make(chan struct{}, 1),
	}

	ws.options = applyOptions(callOptions...)
	ws.websocketConn.SetReadLimit(ws.options.readLimit)
	if ws.options.connectionPool != nil {
		// 通知连接池记录新的连接
		ws.options.connectionPool.Add(ws)
	}

	return ws
}

// Ping 发送 Ping 消息，将会发送 PingMessage（RFC 定义的控制帧 9）和 TextMessage（RFC 定义的控制帧 1）两种类型的 Ping 消息，Ping 消息的内容将会是 PingMessageHeart（"❤️"）
func (w *Websocket) Ping() error {
	err := w.websocketConn.WriteMessage(websocket.TextMessage, []byte(w.options.pingMessageHeart))
	if err != nil {
		return err
	}

	return nil
}

// Write 发送消息，将会发送 TextMessage（RFC 定义的控制帧 1）类型的消息
func (w *Websocket) Write(message []byte) {
	// 如果当前连接已经关闭，则不再发送消息
	if w.alreadyClosed {
		return
	}

	w.writeChannel <- message
}

// WriteJSON 发送 JSON 消息，将会发送 TextMessage（RFC 定义的控制帧 1）类型的消息
// NOTICE: 如果 JSON 序列化失败，将会返回空 JSON 对象（{}）
func (w *Websocket) WriteJSON(message interface{}) {
	jsonData, err := json.Marshal(message)
	if err != nil {
		w.Write([]byte("{}"))
	}

	w.Write(jsonData)
}

// Close 关闭连接，在关闭前将会发送 CloseMessage（RFC 定义的控制帧 8）类型的消息来通知客户端关闭自己的连接
func (w *Websocket) Close() {
	// 如果当前连接已经关闭，则不再重复关闭：乐观锁
	if w.alreadyClosed {
		return
	}

	// 优先设定已经关闭的标记，防止重复关闭
	w.alreadyClosed = true
	// 发送关闭连接消息
	_ = w.websocketConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	// 关闭连接
	_ = w.websocketConn.Close()
	// 通知所有 goroutine 退出
	w.closeChannel <- struct{}{}
	// 完成后关闭 close channel
	close(w.closeChannel)
	// 完成后关闭 write channel
	close(w.writeChannel)
	if w.options.connectionPool != nil {
		// 通知连接池关闭连接
		w.options.connectionPool.Remove(w)
	}
}

// CloseOnError 关闭连接并返回格式化好的错误，当发生错误时将会调用此方法，该方法将会自行调用 Close，不需要再次调用 Close
func (w *Websocket) CloseOnError(err error) {
	err = w.websocketConn.WriteJSON(err)
	if err != nil {
		// TODO: log error
		log.Println(err)
	}

	w.Close()
}

func (w *Websocket) Open() {
	go func() { w.readPump() }()
	go func() { w.writePump() }()
}
