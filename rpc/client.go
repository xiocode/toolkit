package protorpc

import (
	"bufio"
	"code.google.com/p/goprotobuf/proto"
	"errors"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"sync"
)

type clientCodec struct {
	mutex   sync.Mutex
	r       *bufio.Reader
	w       *bufio.Writer
	c       io.Closer
	pending map[uint64]*rpc.Request
	res     *Response
	next    uint64
}

func NewClientCodec(c io.ReadWriteCloser) rpc.ClientCodec {
	return &clientCodec{
		r:       bufio.NewReader(c),
		w:       bufio.NewWriter(c),
		c:       c,
		pending: make(map[uint64]*rpc.Request),
		next:    1,
	}
}

func (c *clientCodec) WriteRequest(rpcreq *rpc.Request, param interface{}) error {
	rr := *rpcreq
	req := &Request{}

	c.mutex.Lock()
	req.Id = proto.Uint64(c.next)
	c.next++
	c.pending[*req.Id] = &rr
	c.mutex.Unlock()

	req.Method = proto.String(rpcreq.ServiceMethod)
	if msg, ok := param.(proto.Message); ok {
		body, err := proto.Marshal(msg)
		if err != nil {
			return err
		}
		req.Body = body
	} else {
		return fmt.Errorf("marshal request param error: %s", param)
	}

	f, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	if err := write(c.w, f); err != nil {
		return err
	}

	return nil
}

func (c *clientCodec) ReadResponseHeader(rpcres *rpc.Response) error {
	f, err := read(c.r)
	if err != nil {
		return err
	}

	res := &Response{}
	if err := proto.Unmarshal(f, res); err != nil {
		return err
	}

	c.mutex.Lock()
	p, ok := c.pending[res.GetId()]
	if !ok {
		c.mutex.Unlock()
		return errors.New("invalid sequence number in response")
	}
	c.res = res
	delete(c.pending, res.GetId())
	c.mutex.Unlock()

	rpcres.Seq = p.Seq
	rpcres.ServiceMethod = p.ServiceMethod
	rpcres.Error = res.GetError()

	return nil
}

func (c *clientCodec) ReadResponseBody(value interface{}) error {
	if value == nil {
		return nil
	}

	if msg, ok := value.(proto.Message); ok {
		if err := proto.Unmarshal(c.res.Body, msg); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unmarshal response body error: %s", value)
	}

	return nil
}

func (c *clientCodec) Close() error {
	return c.c.Close()
}

func NewClient(c io.ReadWriteCloser) *rpc.Client {
	return rpc.NewClientWithCodec(NewClientCodec(c))
}

func Dial(network, address string) (*rpc.Client, error) {
	c, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(c), err
}
