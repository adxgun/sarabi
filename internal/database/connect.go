package database

import (
	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	types2 "sarabi/internal/types"
)

func Open(dir string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dir), &gorm.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to open DB: "+dir)
	}

	if err := db.Debug().AutoMigrate(
		&types2.Application{},
		&types2.Secret{},
		&types2.Deployment{},
		&types2.DeploymentSecret{},
		&types2.Domain{},
		&types2.BackupSettings{},
		&types2.Credential{},
		&types2.Backup{}); err != nil {
		return nil, err
	}

	return db, nil
}
