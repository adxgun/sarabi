//go:build !linux

package firewall

type noOpManager struct{}

func (n noOpManager) BlockPortAccess(port uint) error {
	return nil
}

func (n noOpManager) WhitelistIP(ip string, port uint) error {
	return nil
}

func (n noOpManager) BlacklistIP(ip string, port uint) error {
	return nil
}

func newNoOpManager() Manager {
	return &noOpManager{}
}

func NewManager() Manager {
	return newNoOpManager()
}
