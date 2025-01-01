package storage

const (
	BackupTempDir     = "/var/sarabi/data/backups/tmp"
	BackupDir         = "/var/sarabi/data/backups"
	Path              = "/var/sarabi/data"
	DBDir             = Path + "/floki.db"
	AppLogDir         = Path + "/logs"
	ServerCrtFilePath = Path + "/certs/server.crt"
	ServerKeyFilePath = Path + "/certs/server.key"
)
