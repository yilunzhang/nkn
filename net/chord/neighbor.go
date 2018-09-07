package chord

func (r *Ring) shouldConnectToHost(host string) bool {
	for _, vn := range r.Vnodes {
		if vn != nil && vn.shouldConnectToHost(host) {
			return true
		}
	}
	return false
}

func (vn *localVnode) shouldConnectToHost(host string) bool {
	// successor list
	for _, n := range vn.successors {
		if n != nil && n.Host == host {
			return true
		}
	}

	// predecessor
	if vn.predecessor != nil && vn.predecessor.Host == host {
		return true
	}

	// finger table
	for _, n := range vn.finger {
		if n != nil && n.Host == host {
			return true
		}
	}

	return false
}

func (vn *localVnode) ShouldAddrInNeighbors(addr []byte) bool {
	if CompareId(vn.Id, addr) == 0 {
		return false
	}

	for _, n := range vn.finger {
		if n == nil {
			continue
		}
		if n.Host == vn.Host {
			continue
		}
		if CompareId(n.Id, addr) == 0 {
			return true
		}
	}

	hb := vn.ring.config.hashBits
	offset := powerOffset(addr, fingerId(addr, vn.Id, hb), hb)
	if vn.predecessor != nil && betweenRightIncl(vn.predecessor.Id, vn.Id, offset) {
		return true
	}

	return false
}

func (vn *localVnode) ClosestNeighborIterator(key []byte) (closestPreceedingVnodeIterator, error) {
	cp := closestPreceedingVnodeIterator{}
	cp.init(vn, key, true, true)
	return cp, nil
}

func (vn *localVnode) GetNeighbors() []*Vnode {
	seen := make(map[*Vnode]bool)
	neighbors := []*Vnode{}
	for _, n := range vn.finger {
		if n == nil {
			continue
		}
		if n.Host == vn.Host {
			continue
		}
		if _, value := seen[n]; !value {
			seen[n] = true
			neighbors = append(neighbors, n)
		}
	}
	return neighbors
}

func (vn *localVnode) GetFloodingNeighbors(fromId []byte) []*Vnode {
	neighbors := []*Vnode{}
	maxIdx := fingerId(fromId, vn.Id, vn.ring.config.hashBits)
	for i := maxIdx - 1; i >= 0; i-- {
		if vn.finger[i] == nil {
			continue
		}
		if vn.finger[i].Host == vn.Host {
			continue
		}
		if vn.finger[i] != vn.finger[i+1] {
			neighbors = append(neighbors, vn.finger[i])
		}
	}
	return neighbors
}
