package firewall

type (
	// Manager is the firewall manager that manages application databases connection restriction.
	// the implementation uses nftables which is only available on Linux. A later support will come for non linux OSes.
	// currently, the compiler chooses the interface implementation based on build tags.
	// Current behavior: on Mac/Windows -> Does nothing. On Linux -> interact with nftables to carry out commands
	Manager interface {
		BlockPortAccess(port uint) error
		WhitelistIP(ip string, port uint) error
		BlacklistIP(ip string, port uint) error
	}
)
