package ipx

import (
	"errors"
	"net"
	"sort"
)

// Collapse combines subnets into their closest available parent. All networks must of the same type.
func Collapse(toMerge []*net.IPNet) []*net.IPNet {
	if len(toMerge) == 0 {
		return nil
	}
	four := toMerge[0].IP.To4() != nil
	for _, ipN := range toMerge[1:] {
		if four != (ipN.IP.To4() != nil) {
			panic(errors.New("all versions must be the same"))
		}
	}
	if four {
		return collapse4(toMerge)
	}
	return collapse6(toMerge)
}

func collapse4(toMerge []*net.IPNet) []*net.IPNet {
	nets := make([]ip4Net, 0, len(toMerge))
	for _, m := range toMerge {
		nets = append(nets, newIP4Net(m))
	}

	supers := make(map[ip4Net]ip4Net)
	for len(nets) > 0 {
		n := nets[len(nets)-1]
		nets = nets[:len(nets)-1]

		s := n.super()
		other, ok := supers[s]
		if !ok {
			supers[s] = n
			continue
		}
		if other == n {
			continue
		}

		// we have found two nets with same immediate parent -- merge 'em
		delete(supers, s)
		nets = append(nets, s)
	}

	merged := make(ip4Nets, 0, len(supers))
	for _, v := range supers {
		merged = append(merged, v)
	}
	sort.Sort(merged)

	result := []*net.IPNet{merged[0].asNet()}
	lastMask := merged[0].mask()
	lastAddr := merged[0].addr
	for _, m := range merged[1:] {
		if lastAddr == m.addr&lastMask {
			continue
		}
		result = append(result, m.asNet())
		lastMask, lastAddr = m.mask(), m.addr
	}
	return result
}

func collapse6(toMerge []*net.IPNet) []*net.IPNet {
	nets := make([]ip6Net, 0, len(toMerge))
	for _, m := range toMerge {
		nets = append(nets, newIP6Net(m))
	}

	supers := make(map[ip6Net]ip6Net)
	for len(nets) > 0 {
		n := nets[len(nets)-1]
		nets = nets[:len(nets)-1]

		s := n.super()
		other, ok := supers[s]
		if !ok {
			supers[s] = n
			continue
		}
		if other == n {
			continue
		}

		// we have found two nets with same immediate parent -- merge 'em
		delete(supers, s)
		nets = append(nets, s)
	}

	merged := make(ip6Nets, 0, len(supers))
	for _, v := range supers {
		merged = append(merged, v)
	}
	sort.Sort(merged)

	result := []*net.IPNet{merged[0].asNet()}
	lastMask := merged[0].mask()
	lastAddr := merged[0].addr
	for _, m := range merged[1:] {
		if lastAddr == m.addr.And(lastMask) {
			continue
		}
		result = append(result, m.asNet())
		lastMask, lastAddr = m.mask(), m.addr
	}
	return result
}

type ip4Net struct {
	addr   uint32
	prefix uint8
}

func newIP4Net(ipN *net.IPNet) ip4Net {
	ones, _ := ipN.Mask.Size()
	return ip4Net{to32(ipN.IP), uint8(ones)}
}

func (n ip4Net) super() ip4Net {
	n.addr &^= 1 << (32 - n.prefix) // unset last bit of net mask to find supernet address
	n.prefix--
	return n
}

func (n ip4Net) asNet() *net.IPNet {
	r := &net.IPNet{IP: make(net.IP, 4), Mask: make(net.IPMask, 4)}
	from32(n.addr, r.IP)
	from32(n.mask(), r.Mask) // set first eight bits
	return r
}

func (n ip4Net) mask() uint32 {
	return ^(1<<(32-n.prefix) - 1)
}

type ip4Nets []ip4Net

func (n ip4Nets) Len() int {
	return len(n)
}

func (n ip4Nets) Less(i, j int) bool {
	return n[i].addr < n[j].addr
}

func (n ip4Nets) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

type ip6Net struct {
	addr   uint128
	prefix uint8
}

func newIP6Net(ipN *net.IPNet) ip6Net {
	ones, _ := ipN.Mask.Size()
	return ip6Net{to128(ipN.IP), uint8(ones)}
}

func (n ip6Net) super() ip6Net {
	// unset last bit of net mask to find supernet address
	n.addr = n.addr.And(uint128{0, 1}.Lsh(128 - uint(n.prefix)).Not())
	n.prefix--
	return n
}

func (n ip6Net) asNet() *net.IPNet {
	r := &net.IPNet{IP: make(net.IP, 16), Mask: make(net.IPMask, 16)}
	from128(n.addr, r.IP)
	from128(n.mask(), r.Mask) // set prefix bits
	return r
}

func (n ip6Net) mask() uint128 {
	return uint128{0, 1}.Lsh(128 - uint(n.prefix)).Minus(uint128{0, 1}).Not()
}

type ip6Nets []ip6Net

func (n ip6Nets) Len() int {
	return len(n)
}

func (n ip6Nets) Less(i, j int) bool {
	return n[i].addr.Cmp(n[j].addr) == -1
}

func (n ip6Nets) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}
