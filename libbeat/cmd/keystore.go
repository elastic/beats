package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	tml "golang.org/x/crypto/ssh/terminal"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common/cli"
	"github.com/elastic/beats/libbeat/common/terminal"
	"github.com/elastic/beats/libbeat/keystore"
)

func getKeystore(name, version string) (keystore.Keystore, error) {
	b, err := instance.NewBeat(name, "", version)

	if err != nil {
		return nil, fmt.Errorf("error initializing beat: %s", err)
	}

	if err = b.Init(); err != nil {
		return nil, fmt.Errorf("error initializing beat: %s", err)
	}

	return b.Keystore(), nil
}

// genKeystoreCmd initialize the Keystore command to manage the Keystore
// with the following subcommands:
//  - create
//  - add
//  - remove
//  - list
func genKeystoreCmd(
	name, idxPrefix, version string,
	beatCreator beat.Creator,
	runFlags *pflag.FlagSet,
) *cobra.Command {
	keystoreCmd := cobra.Command{
		Use:   "keystore",
		Short: "Manage secrets keystore",
	}

	keystoreCmd.AddCommand(genCreateKeystoreCmd(name, version))
	keystoreCmd.AddCommand(genAddKeystoreCmd(name, version))
	keystoreCmd.AddCommand(genRemoveKeystoreCmd(name, version))
	keystoreCmd.AddCommand(genListKeystoreCmd(name, version))

	return &keystoreCmd
}

func genCreateKeystoreCmd(name, version string) *cobra.Command {
	var flagForce bool
	command := &cobra.Command{
		Use:   "create",
		Short: "Create keystore",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			return createKeystore(name, version, flagForce)
		}),
	}
	command.Flags().BoolVar(&flagForce, "force", false, "override the existing keystore")
	return command
}

func genAddKeystoreCmd(name, version string) *cobra.Command {
	var flagForce bool
	var flagStdin bool
	command := &cobra.Command{
		Use:   "add",
		Short: "Add secret",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			store, err := getKeystore(name, version)
			if err != nil {
				return err
			}
			return addKey(store, args, flagForce, flagStdin)
		}),
	}
	command.Flags().BoolVar(&flagStdin, "stdin", false, "Use the stdin as the source of the secret")
	command.Flags().BoolVar(&flagForce, "force", false, "Override the existing key")
	return command
}

func genRemoveKeystoreCmd(name, version string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "remove secret",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			store, err := getKeystore(name, version)
			if err != nil {
				return err
			}
			return removeKey(store, args)
		}),
	}
}

func genListKeystoreCmd(name, version string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List keystore",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			store, err := getKeystore(name, version)
			if err != nil {
				return err
			}
			return list(store)
		}),
	}
}

func createKeystore(name, version string, force bool) error {
	store, err := getKeystore(name, version)
	if err != nil {
		return err
	}

	if store.IsPersisted() == true && force == false {
		response := terminal.PromptYesNo("A keystore already exists, Overwrite?", true)
		if response == true {
			err := store.Create(true)
			if err != nil {
				return fmt.Errorf("error creating the keystore: %s", err)
			}
		} else {
			fmt.Println("Exiting without creating keystore.")
			return nil
		}
	} else {
		err := store.Create(true)
		if err != nil {
			return fmt.Errorf("Error creating the keystore: %s", err)
		}
	}
	fmt.Printf("Created %s keystore\n", name)
	return nil
}

func addKey(store keystore.Keystore, keys []string, force, stdin bool) error {
	if len(keys) == 0 {
		return errors.New("failed to create the secret: no key provided")
	}

	if len(keys) > 1 {
		return fmt.Errorf("could not create secret for: %s, you can only provide one key per invocation", keys)
	}

	if store.IsPersisted() == false {
		if force == false {
			answer := terminal.PromptYesNo("The keystore does not exist. Do you want to create it?", false)
			if answer == false {
				return errors.New("exiting without creating keystore")
			}
		}
		err := store.Create(true)
		if err != nil {
			return fmt.Errorf("could not create keystore, error: %s", err)
		}
		fmt.Println("Created keystore")
	}

	key := strings.TrimSpace(keys[0])
	value, err := store.Retrieve(key)
	if value != nil && force == false {
		if stdin == true {
			return fmt.Errorf("the settings %s already exist in the keystore use `--force` to replace it", key)
		}
		answer := terminal.PromptYesNo(fmt.Sprintf("Setting %s already exists, Overwrite?", key), false)
		if answer == false {
			fmt.Println("Exiting without modifying keystore.")
			return nil
		}
	}

	var keyValue []byte
	if stdin {
		reader := bufio.NewReader(os.Stdin)
		keyValue, err = ioutil.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("could not read input from stdin")
		}
	} else {
		fmt.Printf("Enter value for %s: ", key)
		keyValue, err = tml.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("could not read value from the input, error: %s", err)
		}
	}
	if err = store.Store(key, keyValue); err != nil {
		return fmt.Errorf("could not add the key in the keystore, error: %s", err)
	}
	if err = store.Save(); err != nil {
		return fmt.Errorf("fail to save the keystore: %s", err)
	} else {
		fmt.Println("Successfully updated the keystore")
	}
	return nil
}

func removeKey(store keystore.Keystore, keys []string) error {
	if len(keys) == 0 {
		return errors.New("you must supply at least one key to remove")
	}

	if store.IsPersisted() == false {
		return errors.New("the keystore doesn't exist. Use the 'create' command to create one")
	}

	for _, key := range keys {
		key = strings.TrimSpace(key)
		_, err := store.Retrieve(key)
		if err != nil {
			return fmt.Errorf("could not find key '%v' in the keystore", key)
		}

		store.Delete(key)
		err = store.Save()
		if err != nil {
			return fmt.Errorf("could not update the keystore with the changes, key: %s, error: %v", key, err)
		}
		fmt.Printf("successfully removed key: %s\n", key)
	}
	return nil
}

func list(store keystore.Keystore) error {
	keys, err := store.List()
	if err != nil {
		return fmt.Errorf("could not read values from the keystore, error: %s", err)
	}
	for _, key := range keys {
		fmt.Println(key)
	}
	return nil
}
