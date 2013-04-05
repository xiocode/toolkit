package protorpc

import (
	"bufio"
	"code.google.com/p/goprotobuf/proto"
	"errors"
	"fmt"
	"io"
	"net/rpc"
	"sync"
)

type serverCodec struct {
	mutex   sync.Mutex
	r       *bufio.Reader
	w       *bufio.Writer
	c       io.Closer
	pending map[uint64]uint64
	req     *Request
	seq     uint64
}

func NewServerCodec(c io.ReadWriteCloser) rpc.ServerCodec {
	return &serverCodec{
		r:       bufio.NewReader(c),
		w:       bufio.NewWriter(c),
		c:       c,
		pending: make(map[uint64]uint64),
	}
}

func (c *serverCodec) ReadRequestHeader(rpcreq *rpc.Request) error {
	f, err := read(c.r)
	if err != nil {
		return err
	}

	req := &Request{}
	if err := proto.Unmarshal(f, req); err != nil {
		return err
	}

	rpcreq.ServiceMethod = req.GetMethod()

	c.mutex.Lock()
	rpcreq.Seq = c.seq
	c.seq++
	c.req = req
	c.pending[rpcreq.Seq] = req.GetId()
	c.mutex.Unlock()

	return nil
}

func (c *serverCodec) ReadRequestBody(value interface{}) error {
	if value == nil {
		return nil
	}

	if msg, ok := value.(proto.Message); ok {
		if err := proto.Unmarshal(c.req.GetBody(), msg); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unmarshal request body error: %s", value)
	}

	return nil
}

func (c *serverCodec) WriteResponse(rpcres *rpc.Response, value interface{}) error {
	c.mutex.Lock()
	id, ok := c.pending[rpcres.Seq]
	if !ok {
		c.mutex.Unlock()
		return errors.New("invalid sequence number in response")
	}
	delete(c.pending, rpcres.Seq)
	c.mutex.Unlock()

	res := &Response{}
	res.Id = proto.Uint64(id)

	if rpcres.Error == "" {
		if msg, ok := value.(proto.Message); ok {
			body, err := proto.Marshal(msg)
			if err != nil {
				return err
			}
			res.Body = body
		} else {
			return fmt.Errorf("marshal response body error: %s", value)
		}
	} else {
		res.Error = proto.String(rpcres.Error)
	}

	f, err := proto.Marshal(res)
	if err != nil {
		return err
	}

	if err := write(c.w, f); err != nil {
		return err
	}

	return nil
}

func (c *serverCodec) Close() error {
	return c.c.Close()
}

func ServeConn(c io.ReadWriteCloser) {
	rpc.ServeCodec(NewServerCodec(c))
}
