package client

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/ubuntu/zsys"
	"github.com/ubuntu/zsys/internal/config"
	"github.com/ubuntu/zsys/internal/streamlogger"
	"github.com/ubuntu/zsys/internal/zfs"
)

var (
	userdataCmd = &cobra.Command{
		Use:    "userdata COMMAND",
		Short:  "User datasets creation and renaming",
		Hidden: true,
		RunE:   requireSubcommand,
	}
	userdataCreateCmd = &cobra.Command{
		Use:   "create USER HOME_DIRECTORY",
		Short: "Create a new home user dataset via an user dataset (if doesn't exist) creation",
		Args:  cobra.ExactArgs(2),
		Run:   func(cmd *cobra.Command, args []string) { cmdErr = createUserData(args[0], args[1]) },
	}
	userdataRenameCmd = &cobra.Command{
		Use:   "set-home OLD_HOME NEW_HOME",
		Short: "Rename a user's home directory via renaming corresponding user dataset",
		Args:  cobra.ExactArgs(2),
		Run:   func(cmd *cobra.Command, args []string) { cmdErr = changeHomeOnUserData(args[0], args[1]) },
	}
)

func init() {
	rootCmd.AddCommand(userdataCmd)
	userdataCmd.AddCommand(userdataCreateCmd)
	userdataCmd.AddCommand(userdataRenameCmd)
}

// createUserData creates a new userdata for user and set it to homepath on current zsys system.
// if the user already exists for a dataset attached to the current system, set its mountpoint to homepath.
func createUserData(user, homepath string) (err error) {
	client, err := newClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(client.Ctx, zsys.DefaultTimeout)
	defer cancel()

	stream, err := client.CreateUserData(ctx, &zsys.CreateUserDataRequest{User: user, Homepath: homepath})
	if err = checkConn(err); err != nil {
		return err
	}

	for {
		_, err := stream.Recv()
		if err == streamlogger.ErrLogMsg {
			continue
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// changeHomeOnUserData change from home to newHome on current zsys system.
func changeHomeOnUserData(home, newHome string) (err error) {
	ms, err := getMachines(context.Background(), zfs.New(context.Background()))
	if err != nil {
		return err
	}

	z := zfs.New(context.Background(), zfs.WithTransactions())
	defer func() {
		if err != nil {
			z.Cancel()
			err = fmt.Errorf("couldn't change home userdataset for %q: "+config.ErrorFormat, home, err)
		} else {
			z.Done()
		}
	}()

	return ms.ChangeHomeOnUserData(context.Background(), home, newHome, z)
}