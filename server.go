package main

import (
	"container/list"
	"github.com/ethereum/ethdb-go"
	"github.com/ethereum/ethutil-go"
	"github.com/ethereum/ethwire-go"
	"log"
	"net"
	"sync/atomic"
	"time"
)

func eachPeer(peers *list.List, callback func(*Peer, *list.Element)) {
	// Loop thru the peers and close them (if we had them)
	for e := peers.Front(); e != nil; e = e.Next() {
		if peer, ok := e.Value.(*Peer); ok {
			callback(peer, e)
		}
	}
}

const (
	processReapingTimeout = 60 // TODO increase
)

type Server struct {
	// Channel for shutting down the server
	shutdownChan chan bool
	// DB interface
	//db *ethdb.LDBDatabase
	db *ethdb.MemDatabase
	// Block manager for processing new blocks and managing the block chain
	blockManager *BlockManager
	// The transaction pool. Transaction can be pushed on this pool
	// for later including in the blocks
	txPool *TxPool
	// Peers (NYI)
	peers *list.List
	// Nonce
	Nonce uint64
}

func NewServer() (*Server, error) {
	//db, err := ethdb.NewLDBDatabase()
	db, err := ethdb.NewMemDatabase()
	if err != nil {
		return nil, err
	}

	ethutil.Config.Db = db

	nonce, _ := ethutil.RandomUint64()
	server := &Server{
		shutdownChan: make(chan bool),
		db:           db,
		peers:        list.New(),
		Nonce:        nonce,
	}
	server.txPool = NewTxPool(server)
	server.blockManager = NewBlockManager(server)

	return server, nil
}

func (s *Server) AddPeer(conn net.Conn) {
	peer := NewPeer(conn, s, true)

	if peer != nil {
		s.peers.PushBack(peer)
		peer.Start()

		log.Println("Peer connected ::", conn.RemoteAddr())
	}
}

func (s *Server) ProcessPeerList(addrs []string) {
	for _, addr := range addrs {
		// TODO Probably requires some sanity checks
		s.ConnectToPeer(addr)
	}
}

func (s *Server) ConnectToPeer(addr string) error {
	peer := NewOutboundPeer(addr, s)

	s.peers.PushBack(peer)

	return nil
}

func (s *Server) OutboundPeers() []*Peer {
	// Create a new peer slice with at least the length of the total peers
	outboundPeers := make([]*Peer, s.peers.Len())
	length := 0
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		if !p.inbound {
			outboundPeers[length] = p
			length++
		}
	})

	return outboundPeers[:length]
}

func (s *Server) InboundPeers() []*Peer {
	// Create a new peer slice with at least the length of the total peers
	inboundPeers := make([]*Peer, s.peers.Len())
	length := 0
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		if p.inbound {
			inboundPeers[length] = p
			length++
		}
	})

	return inboundPeers[:length]
}

func (s *Server) Broadcast(msgType ethwire.MsgType, data []byte) {
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		p.QueueMessage(ethwire.NewMessage(msgType, data))
	})
}

func (s *Server) ReapDeadPeers() {
	for {
		eachPeer(s.peers, func(p *Peer, e *list.Element) {
			if atomic.LoadInt32(&p.disconnect) == 1 || (p.inbound && (time.Now().Unix()-p.lastPong) > int64(5*time.Minute)) {
				log.Println("Dead peer found .. reaping")

				s.peers.Remove(e)
			}
		})

		time.Sleep(processReapingTimeout * time.Second)
	}
}

// Start the server
func (s *Server) Start() {
	// For now this function just blocks the main thread
	ln, err := net.Listen("tcp", ":12345")
	if err != nil {
		// This is mainly for testing to create a "network"
		if Debug {
			log.Println("Connection listening disabled. Acting as client")

			err = s.ConnectToPeer("localhost:12345")
			if err != nil {
				log.Println("Error starting server", err)

				s.Stop()
			}
		} else {
			log.Fatal(err)
		}
	} else {
		// Starting accepting connections
		go func() {
			for {
				conn, err := ln.Accept()
				if err != nil {
					log.Println(err)

					continue
				}

				go s.AddPeer(conn)
			}
		}()
	}

	// Start the reaping processes
	go s.ReapDeadPeers()

	// Start the tx pool
	s.txPool.Start()

	// TMP
	/*
		go func() {
			for {
				s.Broadcast("block", s.blockManager.bc.GenesisBlock().RlpEncode())

				time.Sleep(1000 * time.Millisecond)
			}
		}()
	*/
}

func (s *Server) Stop() {
	// Close the database
	defer s.db.Close()

	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		p.Stop()
	})

	s.shutdownChan <- true

	s.txPool.Stop()
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Server) WaitForShutdown() {
	<-s.shutdownChan
}
