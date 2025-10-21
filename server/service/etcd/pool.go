package etcd

import (
	"errors"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
)

type ConnPool struct {
	conns  map[string]*grpc.ClientConn // key: address
	mu     sync.RWMutex
	addrs  []string
	rrIdx  uint64
	dialFn func(addr string) (*grpc.ClientConn, error)
}

func NewConnPool(dialFn func(addr string) (*grpc.ClientConn, error)) *ConnPool {
	return &ConnPool{
		conns:  make(map[string]*grpc.ClientConn),
		dialFn: dialFn,
	}
}

// UpdateAddresses sets the current address list (called by service discovery)
func (p *ConnPool) UpdateAddresses(addrs []string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// add new
	existing := map[string]struct{}{}
	for _, a := range addrs {
		existing[a] = struct{}{}
		if _, ok := p.conns[a]; !ok {
			conn, err := p.dialFn(a)
			if err == nil {
				p.conns[a] = conn
			}
			// if dial fails, we still keep trying next time discovery updates
		}
	}
	// remove old
	for k, c := range p.conns {
		if _, ok := existing[k]; !ok {
			_ = c.Close()
			delete(p.conns, k)
		}
	}
	// update addrs slice
	p.addrs = addrs
}

// PickConn round-robin pick
func (p *ConnPool) PickConn() (*grpc.ClientConn, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.addrs) == 0 {
		return nil, errors.New("no available backends")
	}
	idx := int(atomic.AddUint64(&p.rrIdx, 1) % uint64(len(p.addrs)))
	addr := p.addrs[idx]
	conn, ok := p.conns[addr]
	if !ok || conn == nil {
		// try to dial on-the-fly
		p.mu.RUnlock()
		p.mu.Lock()
		c, err := p.dialFn(addr)
		if err != nil {
			p.mu.Unlock()
			p.mu.RLock()
			return nil, err
		}
		p.conns[addr] = c
		p.mu.Unlock()
		p.mu.RLock()
		conn = c
	}
	return conn, nil
}

// Close close all conns
func (p *ConnPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.conns {
		_ = c.Close()
	}
	p.conns = map[string]*grpc.ClientConn{}
	p.addrs = nil
}
