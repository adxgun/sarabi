//go:build linux

package firewall

import (
	"encoding/binary"
	"fmt"
	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
	"net"
)

type managerLinux struct {
	conn  *nftables.Conn
	table *nftables.Table
	chain *nftables.Chain
}

func NewManager() Manager {
	table := &nftables.Table{
		Name:   tableName,
		Family: nftables.TableFamilyIPv4,
	}
	chain := &nftables.Chain{
		Name:     chainName,
		Table:    table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookInput,
		Priority: nftables.ChainPriorityFilter,
	}
	return &managerLinux{
		conn:  &nftables.Conn{},
		table: table,
		chain: chain,
	}
}

func (m managerLinux) BlockPortAccess(port uint) error {
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(port))

	rule := &nftables.Rule{
		Table: m.table,
		Chain: m.chain,
		Exprs: []expr.Any{
			&expr.Meta{
				Key:      expr.MetaKeyL4PROTO,
				Register: 1,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.IPPROTO_TCP},
			},
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     portBytes,
			},
			&expr.Verdict{Kind: expr.VerdictDrop},
		},
	}

	m.conn.AddRule(rule)
	if err := m.conn.Flush(); err != nil {
		return fmt.Errorf("failed to block port %d: %w", port, err)
	}
	return nil
}

func (m managerLinux) WhitelistIP(ip string, port uint) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(port))

	rule := &nftables.Rule{
		Table: m.table,
		Chain: m.chain,
		Exprs: []expr.Any{
			&expr.Meta{
				Key:      expr.MetaKeyL4PROTO,
				Register: 1,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.IPPROTO_TCP},
			},
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     portBytes,
			},
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       12,
				Len:          4,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte(net.ParseIP(ip).To4()),
			},
			&expr.Verdict{Kind: expr.VerdictAccept},
		},
	}

	m.conn.AddRule(rule)
	if err := m.conn.Flush(); err != nil {
		return fmt.Errorf("failed to whitelist IP %s for port %d: %w", ip, port, err)
	}
	return nil
}

func (m managerLinux) BlacklistIP(ip string, port uint) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	rules, err := m.conn.GetRules(m.table, m.chain)
	if err != nil {
		return fmt.Errorf("failed to retrieve rules: %w", err)
	}

	for _, rule := range rules {
		for _, nextExpr := range rule.Exprs {
			if cmp, ok := nextExpr.(*expr.Cmp); ok && len(cmp.Data) == 4 && cmp.Op == expr.CmpOpEq {
				if string(cmp.Data) == string(net.ParseIP(ip).To4()) {
					err := m.conn.DelRule(rule)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	if err := m.conn.Flush(); err != nil {
		return fmt.Errorf("failed to blacklist IP %s for port %d: %w", ip, port, err)
	}
	return nil
}
