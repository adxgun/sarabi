package database

import (
	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"sarabi/internal/types"
)

func Open(dir string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dir), &gorm.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to open DB: "+dir)
	}

	if err := db.AutoMigrate(
		&types.Application{},
		&types.Secret{},
		&types.Deployment{},
		&types.DeploymentSecret{},
		&types.Domain{},
		&types.BackupSettings{},
		&types.ServerConfig{},
		&types.Backup{},
		&types.NetworkAccess{},
		&types.Log{}); err != nil {
		return nil, err
	}

	return db, nil
}
