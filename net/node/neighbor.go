package node

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/nknorg/nkn/net/chord"
	. "github.com/nknorg/nkn/net/protocol"
	"github.com/nknorg/nkn/util/log"
)

// The neighbor node list
type nbrNodes struct {
	sync.RWMutex
	List          map[uint64]*node
	FloodingTable map[uint64][]Noder
}

func (nm *nbrNodes) Broadcast(buf []byte) {
	nm.RLock()
	defer nm.RUnlock()
	for _, node := range nm.List {
		if node.GetState() == ESTABLISH && node.relay == true {
			node.Tx(buf)
		}
	}
}

func (nm *nbrNodes) NodeExisted(uid uint64) bool {
	_, ok := nm.List[uid]
	return ok
}

func (nm *node) AddNbrNode(n Noder) {
	nm.nbrNodes.Lock()
	defer nm.nbrNodes.Unlock()

	if nm.NodeExisted(n.GetID()) {
		fmt.Printf("Insert a existed node\n")
	} else {
		node, err := n.(*node)
		if err == false {
			fmt.Println("Convert the noder error when add node")
			return
		}
		nm.nbrNodes.List[n.GetID()] = node
	}
}

func (nm *node) DelNbrNode(id uint64) (Noder, bool) {
	nm.nbrNodes.Lock()
	defer nm.nbrNodes.Unlock()

	n, ok := nm.nbrNodes.List[id]
	if ok == false {
		return nil, false
	}

	delete(nm.nbrNodes.List, id)

	return n, true
}

func (nm *nbrNodes) GetConnectionCnt() uint {
	nm.RLock()
	defer nm.RUnlock()

	var cnt uint
	for _, node := range nm.List {
		if node.GetState() == ESTABLISH {
			cnt++
		}
	}
	return cnt
}

func (nm *nbrNodes) init() {
	nm.List = make(map[uint64]*node)
}

func (nm *nbrNodes) NodeEstablished(id uint64) bool {
	nm.RLock()
	defer nm.RUnlock()

	n, ok := nm.List[id]
	if ok == false {
		return false
	}

	if n.state != ESTABLISH {
		return false
	}

	return true
}

func (node *node) GetNeighborAddrs() ([]NodeAddr, uint64) {
	node.nbrNodes.RLock()
	defer node.nbrNodes.RUnlock()

	var i uint64
	var addrs []NodeAddr
	for _, n := range node.nbrNodes.List {
		if n.GetState() != ESTABLISH {
			continue
		}
		var addr NodeAddr
		addr.IpAddr, _ = n.GetAddr16()
		addr.Time = n.GetTime()
		addr.Services = n.Services()
		addr.Port = n.GetPort()
		addr.ID = n.GetID()
		addrs = append(addrs, addr)

		i++
	}

	return addrs, i
}

func (node *node) GetNeighborHeights() ([]uint32, uint64) {
	node.nbrNodes.RLock()
	defer node.nbrNodes.RUnlock()

	var i uint64
	heights := []uint32{}
	for _, n := range node.nbrNodes.List {
		if n.GetState() == ESTABLISH {
			height := n.GetHeight()
			heights = append(heights, height)
			i++
		}
	}
	return heights, i
}

func (node *node) GetActiveNeighbors() []Noder {
	node.nbrNodes.RLock()
	defer node.nbrNodes.RUnlock()

	nodes := []Noder{}
	for _, n := range node.nbrNodes.List {
		if n.GetState() != INACTIVITY {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func (node *node) GetNeighborNoder() []Noder {
	node.nbrNodes.RLock()
	defer node.nbrNodes.RUnlock()

	nodes := []Noder{}
	for _, n := range node.nbrNodes.List {
		if n.GetState() == ESTABLISH {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func (node *node) GetSyncFinishedNeighbors() []Noder {
	node.nbrNodes.RLock()
	defer node.nbrNodes.RUnlock()

	var nodes []Noder
	for _, n := range node.nbrNodes.List {
		if n.GetState() == ESTABLISH && n.GetSyncState() == PersistFinished {
			nodes = append(nodes, n)
		}
	}

	return nodes
}

func (node *node) GetNbrNodeCnt() uint32 {
	node.nbrNodes.RLock()
	defer node.nbrNodes.RUnlock()
	var count uint32
	for _, n := range node.nbrNodes.List {
		if n.GetState() == ESTABLISH {
			count++
		}
	}
	return count
}

func (node *node) GetNeighborByAddr(addr string) Noder {
	node.nbrNodes.RLock()
	defer node.nbrNodes.RUnlock()
	for _, n := range node.nbrNodes.List {
		if n.GetState() == ESTABLISH {
			if n.GetAddrStr() == addr {
				return n
			}
		}
	}
	return nil
}

func (node *node) GetNeighborByChordAddr(chordAddr []byte) Noder {
	node.nbrNodes.RLock()
	defer node.nbrNodes.RUnlock()
	for _, n := range node.nbrNodes.List {
		if n.GetState() == ESTABLISH {
			if bytes.Compare(n.GetChordAddr(), chordAddr) == 0 {
				return n
			}
		}
	}
	return nil
}

func (node *node) IsAddrInNeighbors(addr string) bool {
	neighbor := node.GetNeighborByAddr(addr)
	if neighbor != nil {
		return true
	}
	return false
}

func (node *node) IsChordAddrInNeighbors(chordAddr []byte) bool {
	neighbor := node.GetNeighborByChordAddr(chordAddr)
	if neighbor != nil {
		return true
	}
	return false
}

func (node *node) ShouldChordAddrInNeighbors(addr []byte) (bool, error) {
	chordNode, err := node.ring.GetFirstVnode()
	if err != nil {
		return false, err
	}
	if chordNode == nil {
		return false, errors.New("No chord node binded")
	}
	return chordNode.ShouldAddrInNeighbors(addr), nil
}

func (n *node) getFloodingNeighbors(from Noder) []Noder {
	var nodeNbrs []Noder
	chordNode, err := n.ring.GetFirstVnode()
	if err != nil {
		log.Error("Get chord node error:", err)
		return nodeNbrs
	}

	var chordNbrs []*chord.Vnode
	if from == nil {
		chordNbrs = chordNode.GetNeighbors()
	} else {
		chordNbrs = chordNode.GetFloodingNeighbors(from.GetChordAddr())
	}

	n.nbrNodes.RLock()
	defer n.nbrNodes.RUnlock()

	for _, chordNbr := range chordNbrs {
		nodeNbr := n.GetNeighborByChordAddr(chordNbr.Id)
		if nodeNbr != nil {
			nodeNbrs = append(nodeNbrs, nodeNbr)
		}
	}

	return nodeNbrs
}

func (n *node) RecomputeFloodingTable() {
	floodingTable := make(map[uint64][]Noder)
	for _, nbr := range n.nbrNodes.List {
		floodingTable[nbr.GetID()] = n.getFloodingNeighbors(nbr)
	}

	n.nbrNodes.Lock()
	n.nbrNodes.FloodingTable = floodingTable
	n.nbrNodes.Unlock()
}

func (n *node) GetFloodingNeighbors(from Noder) []Noder {
	if from == nil {
		return n.getFloodingNeighbors(nil)
	}
	n.nbrNodes.RLock()
	defer n.nbrNodes.RUnlock()
	nbrs, _ := n.FloodingTable[from.GetID()]
	return nbrs
}
