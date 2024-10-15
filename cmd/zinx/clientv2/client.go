package clientv2

import (
	"context"
	"errors"
	"time"

	"github.com/aceld/zinx/ziface"
	"github.com/aceld/zinx/zlog"
	"github.com/aceld/zinx/znet"
)

type Client struct {
	ziface.IClient
	ctx          context.Context
	cancel       context.CancelFunc
	disconnectCh chan struct{}
	connectOkCh  chan struct{}
}

func NewClient(host string, port int) *Client {
	c := &Client{
		disconnectCh: make(chan struct{}, 1),
		connectOkCh:  make(chan struct{}, 1),
	}
	c.IClient = znet.NewClient(host, port)

	c.SetOnConnStart(c.onConnStart)       //默认设置连接成功的回调是往connectOkCh中发送消息
	c.IClient.SetOnConnStop(c.onConnStop) //默认设置断开连接的回调是往disconnectCh中发送消息
	return c
}

// 如果业务层设置连接成功的回调函数, 那么包装一下，也把默认的往connectOkCh发通知的回调函数也设置上
func (c *Client) SetOnConnStart(handler func(conn ziface.IConnection)) {
	handlerWrap := func(conn ziface.IConnection) {
		c.onConnStart(conn)
		handler(conn)
	}
	c.IClient.SetOnConnStart(handlerWrap)
}

func (c *Client) onConnStart(conn ziface.IConnection) {
	select {
	case c.connectOkCh <- struct{}{}:
	default:
	}
}

// 如果业务层设置连接断开时的回调函数, 那么包装一下，也把默认的回调函数也设置上
func (c *Client) SetOnConnStop(handler func(conn ziface.IConnection)) {
	handlerWrap := func(conn ziface.IConnection) {
		c.onConnStop(conn)
		handler(conn)
	}
	c.IClient.SetOnConnStop(handlerWrap)
}

func (c *Client) onConnStop(conn ziface.IConnection) {
	select {
	case c.disconnectCh <- struct{}{}:
	default:
	}
}

func (c *Client) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *Client) StartWithContext(ctx context.Context) {
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.IClient.Start()

	go func() {
		for {
			select {
			case <-c.ctx.Done():
				if c.Conn() != nil {
					c.IClient.Stop()
				}
				return
			case <-c.disconnectCh:
				zlog.Errorf("%v->%v, disconnect, reconnect after 1s", c.Conn().LocalAddr(), c.Conn().RemoteAddr())
				time.Sleep(time.Second)
				c.IClient.Restart()
			case err := <-c.GetErrChan():
				zlog.Errorf("dial err:%v, reconnect after 3s", err)
				time.Sleep(time.Second * 3)
				c.IClient.Restart()
			}
		}
	}()
}

// 同步连接服务器，直到连接成功或者取消连接
func (c *Client) Connect(ctx context.Context) error {
	c.StartWithContext(ctx)
	select {
	case <-c.ctx.Done():
		return c.ctx.Err()
	case <-c.connectOkCh:
		return nil
	}
}

// 同步连接服务器，直到连接成功或者连接超时
func (c *Client) ConnectWithTimeout(timeout time.Duration) error {
	t := time.NewTimer(timeout)
	defer t.Stop()

	c.StartWithContext(context.Background())
	select {
	case <-t.C:
		c.Stop()
		return errors.New("connect timeout")
	case <-c.connectOkCh:
		return nil
	}
}
