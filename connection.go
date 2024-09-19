package wso

import (
	"sync"

	"github.com/go-redis/redis"
	"github.com/redis/rueidis"
)

type ConnectionPool struct {
	Clients map[*Websocket]struct{}

	alreadyClosed bool
	close         chan struct{}
	connect       chan *Websocket
	disconnect    chan *Websocket
	broadcast     chan []byte

	options *websocketOptions
}

func NewConnectionPool(callOptions ...WebsocketCallOption) *ConnectionPool {
	cp := &ConnectionPool{
		Clients:    make(map[*Websocket]struct{}),
		close:      make(chan struct{}),
		connect:    make(chan *Websocket),
		disconnect: make(chan *Websocket),
		broadcast:  make(chan []byte),
	}

	cp.options = applyOptions(callOptions...)
	return cp
}

func (p *ConnectionPool) SubscribeRedisPubsub(pubsub *redis.PubSub) *ConnectionPool {
	// 分离 goroutine 来处理广播消息
	go func() {
		// 监听 Redis 的 Pub/Sub 消息频道
		defer func() {
			pubsub.Close()
		}()
		for {
			select {
			case <-p.close:
				p.options.logger.Debug("subscribed pubsub closed")
				return
			case message := <-pubsub.Channel():
				// 广播到所有连接
				p.Broadcast([]byte(message.Payload))
			}
		}
	}()

	return p
}

func (p *ConnectionPool) SubscribeRueidisPubsub() (*ConnectionPool, rueidis.PubSubHooks) {
	return p, rueidis.PubSubHooks{
		OnMessage: func(m rueidis.PubSubMessage) {
			// 广播到所有连接
			p.Broadcast([]byte(m.Message))
		},
	}
}

func (p *ConnectionPool) Open() *ConnectionPool {
	go func() {
		for {
			select {
			case <-p.close:
				p.options.logger.Debug("opened connection pool closed")
				return
			case client := <-p.connect:
				p.Clients[client] = struct{}{}
			case client := <-p.disconnect:
				delete(p.Clients, client)
			case message := <-p.broadcast:
				for client := range p.Clients {
					client.Write(message)
				}
			}
		}
	}()

	return p
}

func (p *ConnectionPool) Close() {
	if p.alreadyClosed {
		return
	}

	p.alreadyClosed = true
	for c := range p.Clients {
		c.Close()
		p.Remove(c)
	}

	p.close <- struct{}{}
	close(p.close)
	close(p.connect)
	close(p.disconnect)
	close(p.broadcast)
}

func (p *ConnectionPool) Add(clientWebsocket *Websocket) {
	p.connect <- clientWebsocket
	p.options.logger.Debugf("websocket 已连接: %s (%d 当前总计)", clientWebsocket.remoteAddr, len(p.Clients)+1)
}

func (p *ConnectionPool) Remove(clientWebsocket *Websocket) {
	p.disconnect <- clientWebsocket
	p.options.logger.Debugf("websocket 已离线: %s (%d 当前总计)", clientWebsocket.remoteAddr, len(p.Clients)-1)
}

func (p *ConnectionPool) Broadcast(message []byte) {
	p.broadcast <- message
	p.options.logger.Debugf("广播消息到 %d 个客户端", len(p.Clients))
}

type ConnectionPoolMap struct {
	mutex               sync.Mutex
	mConnectionPool     map[string]*ConnectionPool
	mConnectionPoolKeys []string
}

func NewConnectionPoolMap() *ConnectionPoolMap {
	return &ConnectionPoolMap{
		mConnectionPool:     make(map[string]*ConnectionPool),
		mConnectionPoolKeys: make([]string, 0),
	}
}

func (m *ConnectionPoolMap) Set(key string, value *ConnectionPool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.mConnectionPool[key] = value
	m.mConnectionPoolKeys = append(m.mConnectionPoolKeys, key)
}

func (m *ConnectionPoolMap) Get(key string) *ConnectionPool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	p, ok := m.mConnectionPool[key]
	if !ok || p == nil {
		return nil
	}

	return p
}

func (m *ConnectionPoolMap) Delete(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.mConnectionPool, key)
}

func (m *ConnectionPoolMap) Keys() []string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.mConnectionPoolKeys
}
