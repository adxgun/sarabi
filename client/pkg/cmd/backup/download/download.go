package download

import (
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"io"
	"os"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
)

func NewDownloadBackupCmd(svc api.Service) *cobra.Command {
	var id string
	var location string
	cmd := &cobra.Command{
		Use:     "download",
		Short:   "Download a backup",
		Long:    "Download a database backup file from the storage. You can use the backup ID to download a specific backup. To see the list of backups, use 'floki backup list'",
		Example: "floki backup download --id <backup_id> --location <location>",
		Run: func(cmd *cobra.Command, args []string) {
			backupID, err := uuid.Parse(id)
			if err != nil {
				cmdutil.PrintE("Invalid backup ID: " + id)
				return
			}

			if location == "" {
				location = backupID.String()
			}

			backupFile, err := os.Create(location)
			if err != nil {
				cmdutil.PrintE("Error creating file: " + err.Error())
				return
			}

			defer func() {
				_ = backupFile.Close()
			}()

			cmdutil.StartLoading("Downloading backup...")
			defer cmdutil.StopLoading()

			backup, err := svc.DownloadBackup(cmd.Context(), backupID)
			if err != nil {
				cmdutil.PrintE("Error downloading backup: " + err.Error())
				return
			}

			defer func() {
				_ = backup.Close()
			}()

			_, err = io.Copy(backupFile, backup)
			if err != nil {
				cmdutil.PrintE("Error writing to file: " + err.Error())
				return
			}

			cmdutil.PrintS("Backup downloaded successfully: " + location)
		},
	}

	cmd.Flags().StringVarP(&id, "id", "i", "", "ID of the backup you want to download")
	cmd.Flags().StringVarP(&location, "location", "l", "", "Location to download the backup file")
	return cmd
}
