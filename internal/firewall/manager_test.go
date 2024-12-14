package firewall

import (
	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"github.com/stretchr/testify/assert"
	"net"
	"runtime"
	"testing"
)

const (
	testTable = "test_fwall_table"
	testChain = "test_fwall_chain"
)

func TestBlockPort(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip()
		return
	}

	firewall := NewManager(testTable, testChain)

	err := firewall.BlockPortAccess(8080)
	assert.Nil(t, err, "Expected no error while blocking port")

	table := &nftables.Table{
		Name:   testTable,
		Family: nftables.TableFamilyIPv4,
	}

	conn := &nftables.Conn{}
	chain := &nftables.Chain{
		Name:     testChain,
		Table:    table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookInput,
		Priority: nftables.ChainPriorityFilter,
	}

	rules, err := conn.GetRules(table, chain)
	assert.Nil(t, err, "Expected no error while retrieving rules")

	var portBlocked bool
	for _, rule := range rules {
		for _, nExpr := range rule.Exprs {
			if cmp, ok := nExpr.(*expr.Cmp); ok && len(cmp.Data) == 2 {
				if cmp.Data[0] == 0x1f && cmp.Data[1] == 0x90 { // 8080 in hex
					portBlocked = true
				}
			}
		}
	}

	assert.True(t, portBlocked, "Port 8080 should be blocked")
	cleanup(testTable, testChain, conn)
}

func TestWhitelistIP(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip()
		return
	}

	firewall := NewManager(testTable, testChain)

	err := firewall.BlockPortAccess(8080)
	assert.Nil(t, err, "Expected no error while blocking port")

	table := &nftables.Table{
		Name:   testTable,
		Family: nftables.TableFamilyIPv4,
	}

	conn := &nftables.Conn{}
	chain := &nftables.Chain{
		Name:     testChain,
		Table:    table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookInput,
		Priority: nftables.ChainPriorityFilter,
	}

	ip := "192.168.1.100"
	port := 8080
	err = firewall.WhitelistIP(ip, uint(port))
	assert.Nil(t, err, "Expected no error while whitelisting IP")

	rules, err := conn.GetRules(table, chain)
	assert.Nil(t, err, "Expected no error while retrieving rules")

	var ipWhitelisted bool
	for _, rule := range rules {
		for _, nExpr := range rule.Exprs {
			if cmp, ok := nExpr.(*expr.Cmp); ok && len(cmp.Data) == 4 {
				if net.IP(cmp.Data).String() == ip {
					ipWhitelisted = true
				}
			}
		}
	}

	assert.True(t, ipWhitelisted, "IP %s should be whitelisted", ip)
	cleanup(testTable, testChain, conn)
}

func TestBlacklistIP(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip()
		return
	}

	firewall := NewManager(testTable, testChain)

	err := firewall.BlockPortAccess(8080)
	assert.Nil(t, err, "Expected no error while blocking port")

	table := &nftables.Table{
		Name:   testTable,
		Family: nftables.TableFamilyIPv4,
	}

	conn := &nftables.Conn{}
	chain := &nftables.Chain{
		Name:     testChain,
		Table:    table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookInput,
		Priority: nftables.ChainPriorityFilter,
	}

	ip := "192.168.1.100"
	port := 8080

	// First, whitelist the IP
	err = firewall.WhitelistIP(ip, uint(port))
	assert.Nil(t, err, "Expected no error while whitelisting IP")

	// Then, blacklist the IP
	err = firewall.BlacklistIP(ip, uint(port))
	assert.Nil(t, err, "Expected no error while blacklisting IP")

	rules, err := conn.GetRules(table, chain)
	assert.Nil(t, err, "Expected no error while retrieving rules")

	var ipBlacklisted bool
	for _, rule := range rules {
		for _, nExpr := range rule.Exprs {
			if cmp, ok := nExpr.(*expr.Cmp); ok && len(cmp.Data) == 4 {
				if net.IP(cmp.Data).String() == ip {
					ipBlacklisted = true
				}
			}
		}
	}

	assert.False(t, ipBlacklisted, "IP %s should be blacklisted", ip)
	cleanup(testTable, testChain, conn)
}

func cleanup(tableName, chainName string, conn *nftables.Conn) {
	table := &nftables.Table{Name: tableName, Family: nftables.TableFamilyIPv4}
	chain := &nftables.Chain{Name: chainName, Table: table}
	rules, _ := conn.GetRules(table, chain)
	for _, rule := range rules {
		_ = conn.DelRule(rule)
	}
	_ = conn.Flush()
}
