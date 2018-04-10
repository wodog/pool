package pool

import (
	"errors"
	"io"
	"sync"
	"time"
)

// Config the pool
type Config struct {
	InitialCap uint
	MaxCap     uint
	Timeout    time.Duration
	Factory    func() (io.Closer, error)
}

type resource struct {
	c    io.Closer
	time time.Time
}

// Pool type
type Pool struct {
	m      sync.Mutex
	res    chan resource
	config Config
	closed bool
}

// ErrPoolClosed pool close
var ErrPoolClosed = errors.New("资源池已经被关闭。")

// ErrPoolConfig pool config
var ErrPoolConfig = errors.New("配置错误")

// New new pool
func New(c Config) (*Pool, error) {
	if c.InitialCap > c.MaxCap {
		return nil, ErrPoolConfig
	}

	p := &Pool{
		res:    make(chan resource, c.MaxCap),
		config: c,
	}

	// 初始化数量
	initialConns := make([]io.Closer, p.config.InitialCap)
	for i := range initialConns {
		c, err := p.Acquire()
		if err != nil {
			return nil, err
		}
		initialConns[i] = c
	}
	for _, c := range initialConns {
		p.Release(c)
	}

	return p, nil
}

// NewDefault new default with factory
func NewDefault(factory func() (io.Closer, error)) (*Pool, error) {
	c := Config{
		InitialCap: 0,
		MaxCap:     100,
		Timeout:    5 * time.Minute,
		Factory:    factory,
	}
	return New(c)
}

// Acquire get connection
func (p *Pool) Acquire() (io.Closer, error) {
	select {
	case r, ok := <-p.res:
		if !ok {
			return nil, ErrPoolClosed
		}
		if time.Now().After(r.time.Add(p.config.Timeout)) {
			return p.Acquire()
		}
		return r.c, nil
	default:
		return p.config.Factory()
	}
}

// Release release resource
func (p *Pool) Release(c io.Closer) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		c.Close()
		return
	}

	select {
	case p.res <- resource{c, time.Now()}:
	default:
		c.Close()
	}
}

// Close close pool
func (p *Pool) Close() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.res)
	for r := range p.res {
		r.c.Close()
	}
}

// Len get pool resources length
func (p *Pool) Len() int {
	return len(p.res)
}
