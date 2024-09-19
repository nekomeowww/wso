package wso

import (
	"time"

	"github.com/sirupsen/logrus"
)

type MessageProcessor interface {
	OnTextMessage(websocket *Websocket, message []byte) error
	OnBinaryMessage(websocket *Websocket, message []byte) error
}

type WebsocketCallOption struct {
	applyFunc func(opt *websocketOptions)
}

type websocketOptions struct {
	logger         *logrus.Logger
	connectionPool *ConnectionPool

	autoReplyPong bool
	// 配置是否跳过 Ping 消息，默认跳过
	onMessageSkipPingMessage bool
	// 配置是否跳过 Pong 消息，默认跳过
	onMessageSkipPongMessage bool

	// 心跳 Ping 消息，默认值为 "❤️"
	pingMessageHeart string
	// 心跳 Pong 消息，默认值为 "💚"
	pongMessageHeart string

	// 配置最大消息大小，默认值为 4MB
	readLimit int64
	// 配置读取消息超时时间，默认值为 1 分钟
	readDeadline time.Duration
	// 配置写消息超时时间，默认值为 1 分钟
	writeDeadline time.Duration
	// 配置 Ping 消息发送间隔，默认值为 20 秒
	pingMessagePeriod time.Duration

	processor MessageProcessor
}

// WithLogger 设置日志记录器
func WithLogger(logger *logrus.Logger) WebsocketCallOption {
	return WebsocketCallOption{
		applyFunc: func(opt *websocketOptions) {
			opt.logger = logger
		},
	}
}

// WithConnectionPool 设置连接池
func WithConnectionPool(pool *ConnectionPool) WebsocketCallOption {
	return WebsocketCallOption{
		applyFunc: func(opt *websocketOptions) {
			opt.connectionPool = pool
		},
	}
}

// WithAutoReplyPong 设置是否自动回复 Pong 消息，默认值为 true
func WithAutoReplyPong(autoReplyPong bool) WebsocketCallOption {
	return WebsocketCallOption{
		applyFunc: func(opt *websocketOptions) {
			opt.autoReplyPong = autoReplyPong
		},
	}
}

// WithOnMessageSkipStrategy 设置是否跳过 Ping 消息和 Pong 消息，默认值为 true
func WithOnMessageSkipStrategy(skipPingMessage, skipPongMessage bool) WebsocketCallOption {
	return WebsocketCallOption{
		applyFunc: func(opt *websocketOptions) {
			opt.onMessageSkipPingMessage = skipPingMessage
			opt.onMessageSkipPongMessage = skipPongMessage
		},
	}
}

// WithReadLimit 设置读取消息的最大长度，超过该长度将会断开连接，单位：字节
func WithReadLimit(limit int64) WebsocketCallOption {
	return WebsocketCallOption{
		applyFunc: func(opt *websocketOptions) {
			opt.readLimit = limit
		},
	}
}

// WithReadDeadline 设置读取消息的超时时间
func WithReadDeadline(deadline time.Duration) WebsocketCallOption {
	return WebsocketCallOption{
		applyFunc: func(opt *websocketOptions) {
			opt.readDeadline = deadline
		},
	}
}

// WithWriteDeadline 设置写消息的超时时间
func WithWriteDeadline(deadline time.Duration) WebsocketCallOption {
	return WebsocketCallOption{
		applyFunc: func(opt *websocketOptions) {
			opt.writeDeadline = deadline
		},
	}
}

// WithPingMessagePeriod 设置 Ping 消息的发送间隔
func WithPingMessagePeriod(period time.Duration) WebsocketCallOption {
	return WebsocketCallOption{
		applyFunc: func(opt *websocketOptions) {
			opt.pingMessagePeriod = period
		},
	}
}

// 注册消息监听器，当收到消息时将会调用所有注册的消息监听器。消息监听器将可以返回一个 error 类型的错误，如果返回的错误不为 nil，将会关闭连接
func WithMessageProcessor(processor MessageProcessor) WebsocketCallOption {
	return WebsocketCallOption{
		applyFunc: func(opt *websocketOptions) {
			opt.processor = processor
		},
	}
}

// applyOptions 应用选项
func applyOptions(callOptions ...WebsocketCallOption) *websocketOptions {
	defaultOptions := &websocketOptions{
		logger:                   logrus.New(),
		onMessageSkipPingMessage: true,
		onMessageSkipPongMessage: true,
		pingMessageHeart:         "❤️",
		pongMessageHeart:         "💚",
		readLimit:                int64(4 * BytesUnitMB),
		readDeadline:             time.Minute * 1,
		writeDeadline:            time.Minute * 1,
		pingMessagePeriod:        time.Second * 20,
	}
	if len(callOptions) == 0 {
		return defaultOptions
	}

	optCopy := &websocketOptions{}
	*optCopy = *defaultOptions
	for _, f := range callOptions {
		f.applyFunc(optCopy)
	}

	return optCopy
}
