package casknet

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"

	"borkshop/cask"
)

// Peer tracks a remote address and controls the flow of outbound messages.
type Peer struct {
	server *Server
	addr   *net.UDPAddr
}

// Store instructs the remote peer to store a block with a given hash until the context expires.
func (p *Peer) Store(ctx context.Context, hash cask.Hash, block *cask.Block) error {
	var buf [1500]byte
	copy(buf[0:4], []byte("stor")[:])
	copy(buf[4:4+cask.HashSize], hash[:])
	copy(buf[4+cask.HashSize:], block[0:block.Size()])

	_, err := p.server.conn.WriteToUDP(buf[:4+cask.HashSize+block.Size()], p.addr)
	if err != nil {
		return err
	}

	return nil
}

// Load instructs the remote peer to send back the block with the given hash.
func (p *Peer) Load(ctx context.Context, hash cask.Hash, block *cask.Block) error {
	var buf [1500]byte
	copy(buf[0:4], []byte("load")[:])
	copy(buf[4:4+cask.HashSize], hash[:])

	_, err := p.server.conn.WriteToUDP(buf[:4+cask.HashSize], p.addr)
	if err != nil {
		return err
	}

	return p.server.Store.Load(ctx, hash, block)
}

// Close closes a peer and blocks until it has flushed.
func (p *Peer) Close(ctx context.Context) error {
	return nil
}

// Server represents a local store and handles messages from remote peers.
type Server struct {
	Addr  string
	Store cask.Store

	conn *net.UDPConn
}

// LocalAddr returns the actual UDP address of the local peer.
func (s *Server) LocalAddr() *net.UDPAddr {
	addr, _ := udpAddr(s.conn.LocalAddr().String())
	return addr
}

// Peer returns a peer for sending messages to a remote address.
//
// Peer must be called after the server has started so that it can use the
// server's connection for outbound messages, and for the return address for
// responses.
func (s *Server) Peer(addr *net.UDPAddr) *Peer {
	return &Peer{
		addr:   addr,
		server: s,
	}
}

// Start opens a connection for sending and receiving messages.
//
// Start blocks until the listening port is available and then
// handles incoming messages in the background.
func (s *Server) Start(ctx context.Context) error {
	addr, err := udpAddr(s.Addr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.conn = conn

	go func() {
		defer conn.Close()
		var buf [1500]byte
		for {
			n, raddr, err := conn.ReadFromUDP(buf[:])
			if err != nil {
				log.Printf("%s\n", err)
				continue
			}
			if err := s.handle(raddr, buf[:n]); err != nil {
				log.Printf("%s\n", err)
				continue
			}
		}
	}()

	return nil
}

func (s *Server) handle(raddr *net.UDPAddr, buf []byte) error {
	ctx := context.TODO()

	if len(buf) < 4 {
		return fmt.Errorf("corrupt message")
	}
	switch string(buf[0:4]) {
	case "stor":
		return s.handleStore(ctx, buf, raddr)
	case "load":
		return s.handleLoad(ctx, buf, raddr)
	}
	return nil
}

func (s *Server) handleStore(ctx context.Context, buf []byte, raddr *net.UDPAddr) error {
	var hash cask.Hash
	copy(hash[:], buf[4:])
	var block cask.Block
	copy(block[:], buf[4+cask.HashSize:])
	return s.Store.Store(ctx, hash, &block)
}

func (s *Server) handleLoad(ctx context.Context, buf []byte, raddr *net.UDPAddr) error {
	var hash cask.Hash
	copy(hash[:], buf[4:])
	var block cask.Block
	err := s.Store.Load(ctx, hash, &block)
	if err != nil {
		return err
	}
	rpeer := s.Peer(raddr)
	return rpeer.Store(ctx, hash, &block)
}

// Stop closes the server's listening connection and blocks until
// all handlers have halted.
func (s *Server) Stop(ctx context.Context) error {
	// TODO close channel or set an atomic
	// TODO wait for serve to exit
	return nil
}

func udpAddr(addr string) (*net.UDPAddr, error) {
	host, portstr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portstr)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(host)
	return &net.UDPAddr{Port: port, IP: ip}, nil
}
