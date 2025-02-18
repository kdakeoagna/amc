// Copyright 2023 The AmazeChain Authors
// This file is part of the AmazeChain library.
//
// The AmazeChain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The AmazeChain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the AmazeChain library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"github.com/amazechain/amc/log"
	"github.com/urfave/cli/v2"
	"os"

	"github.com/amazechain/amc/cmd/utils"

	"github.com/amazechain/amc/accounts"
	"github.com/amazechain/amc/accounts/keystore"
	"github.com/amazechain/amc/common/crypto"
	"github.com/amazechain/amc/conf"
	"github.com/amazechain/amc/internal/node"
)

var (
	walletCommand = &cli.Command{
		Name:      "wallet",
		Usage:     "Manage AmazeChain presale wallets",
		ArgsUsage: "",
		Description: `
    AmazeChain wallet import /path/to/my/presale.wallet

will prompt for your password and imports your ether presale account.
It can be used non-interactively with the --password option taking a
passwordfile as argument containing the wallet password in plaintext.`,
		Subcommands: []*cli.Command{
			{

				Name:      "import",
				Usage:     "Import Ethereum presale wallet",
				ArgsUsage: "<keyFile>",
				Action:    importWallet,
				Flags: []cli.Flag{
					DataDirFlag,
					KeyStoreDirFlag,
					PasswordFileFlag,
					LightKDFFlag,
				},
				Description: `
	AmazeChain wallet [options] /path/to/my/presale.wallet

will prompt for your password and imports your ether presale account.
It can be used non-interactively with the --password option taking a
passwordfile as argument containing the wallet password in plaintext.`,
			},
		},
	}

	accountCommand = &cli.Command{
		Name:  "account",
		Usage: "Manage accounts",
		Description: `

Manage accounts, list all existing accounts, import a private key into a new
account, create a new account or update an existing account.

It supports interactive mode, when you are prompted for password as well as
non-interactive mode where passwords are supplied via a given password file.
Non-interactive mode is only meant for scripted use on test networks or known
safe environments.

Make sure you remember the password you gave when creating a new account (with
either new or import). Without it you are not able to unlock your account.

Note that exporting your key in unencrypted format is NOT supported.

Keys are stored under <DATADIR>/keystore.
It is safe to transfer the entire directory or the individual keys therein
between ethereum nodes by simply copying.

Make sure you backup your keys regularly.`,
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "Print summary of existing accounts",
				Action: accountList,
				Flags: []cli.Flag{
					DataDirFlag,
					KeyStoreDirFlag,
				},
				Description: `
			Print a short summary of all accounts`,
			},
			{
				Name:   "new",
				Usage:  "Create a new account",
				Action: accountCreate,
				Flags: []cli.Flag{
					DataDirFlag,
					KeyStoreDirFlag,
					PasswordFileFlag,
					LightKDFFlag,
				},
				Description: `
    AmazeChain account new

Creates a new account and prints the address.

The account is saved in encrypted format, you are prompted for a password.

You must remember this password to unlock your account in the future.

For non-interactive use the password can be specified with the --password flag:

Note, this is meant to be used for testing only, it is a bad idea to save your
password to file or expose in any other way.
`,
			},
			{
				Name:      "update",
				Usage:     "Update an existing account",
				Action:    accountUpdate,
				ArgsUsage: "<address>",
				Flags: []cli.Flag{
					DataDirFlag,
					KeyStoreDirFlag,
					LightKDFFlag,
				},
				Description: `
    AmazeChain account update <address>

Update an existing account.

The account is saved in the newest version in encrypted format, you are prompted
for a password to unlock the account and another to save the updated file.

This same command can therefore be used to migrate an account of a deprecated
format to the newest format or change the password for an account.

For non-interactive use the password can be specified with the --password flag:

    AmazeChain account update [options] <address>

Since only one password can be given, only format update can be performed,
changing your password is only possible interactively.
`,
			},
			{
				Name:   "import",
				Usage:  "Import a private key into a new account",
				Action: accountImport,
				Flags: []cli.Flag{
					DataDirFlag,
					KeyStoreDirFlag,
					PasswordFileFlag,
					LightKDFFlag,
				},
				ArgsUsage: "<keyFile>",
				Description: `
    AmazeChain account import <keyfile>

Imports an unencrypted private key from <keyfile> and creates a new account.
Prints the address.

The keyfile is assumed to contain an unencrypted private key in hexadecimal format.

The account is saved in encrypted format, you are prompted for a password.

You must remember this password to unlock your account in the future.

For non-interactive use the password can be specified with the -password flag:

    AmazeChain account import [options] <keyfile>

Note:
As you can directly copy your encrypted accounts to another ethereum instance,
this import mechanism is not needed when you transfer an account between
nodes.
`,
			},
		},
	}
)

func importWallet(ctx *cli.Context) error {
	if ctx.Args().Len() != 1 {
		utils.Fatalf("keyfile must be given as the only argument")
	}
	keyfile := ctx.Args().First()
	keyJSON, err := os.ReadFile(keyfile)
	if err != nil {
		utils.Fatalf("Could not read wallet file: %v", err)
	}

	stack, err := node.NewNode(ctx.Context, &DefaultConfig)
	if err != nil {
		return err
	}

	passphrase := utils.GetPassPhraseWithList("", false, 0, MakePasswordList(ctx))

	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	acct, err := ks.ImportPreSaleKey(keyJSON, passphrase)
	if err != nil {
		utils.Fatalf("%v", err)
	}
	fmt.Printf("Address: {%x}\n", acct.Address)
	return nil
}

func accountList(ctx *cli.Context) error {
	cfg := DefaultConfig
	// Load config file.
	if len(cfgFile) > 0 {
		if err := conf.LoadConfigFromFile(cfgFile, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	stack, err := node.NewNode(ctx.Context, &cfg)
	if err != nil {
		return err
	}

	var index int
	for _, wallet := range stack.AccountManager().Wallets() {
		for _, account := range wallet.Accounts() {
			fmt.Printf("Account #%d: {%s} %s\n", index, account.Address, &account.URL)
			index++
		}
	}
	return nil
}

// accountCreate creates a new account into the keystore defined by the CLI flags.
func accountCreate(ctx *cli.Context) error {
	cfg := DefaultConfig
	// Load config file.
	if len(cfgFile) > 0 {
		if err := conf.LoadConfigFromFile(cfgFile, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	keydir, err := cfg.NodeCfg.KeyDirConfig()
	if err != nil {
		utils.Fatalf("Failed to read configuration: %v", err)
	}
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	if cfg.NodeCfg.UseLightweightKDF {
		scryptN = keystore.LightScryptN
		scryptP = keystore.LightScryptP
	}

	password := utils.GetPassPhraseWithList("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, MakePasswordList(ctx))

	account, err := keystore.StoreKey(keydir, password, scryptN, scryptP)

	if err != nil {
		utils.Fatalf("Failed to create account: %v", err)
	}
	fmt.Printf("\nYour new key was generated\n\n")
	fmt.Printf("Public address of the key:   %s\n", account.Address.Hex())
	fmt.Printf("Path of the secret key file: %s\n\n", account.URL.Path)
	fmt.Printf("- You can share your public address with anyone. Others need it to interact with you.\n")
	fmt.Printf("- You must NEVER share the secret key with anyone! The key controls access to your funds!\n")
	fmt.Printf("- You must BACKUP your key file! Without the key, it's impossible to access account funds!\n")
	fmt.Printf("- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!\n\n")
	return nil
}

// tries unlocking the specified account a few times.
func unlockAccount(ks *keystore.KeyStore, address string, i int, passwords []string) (accounts.Account, string) {
	account, err := utils.MakeAddress(ks, address)
	if err != nil {
		utils.Fatalf("Could not list accounts: %v", err)
	}
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)
		password := utils.GetPassPhraseWithList(prompt, false, i, passwords)
		err = ks.Unlock(account, password)
		if err == nil {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return account, password
		}
		if err, ok := err.(*keystore.AmbiguousAddrError); ok {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return ambiguousAddrRecovery(ks, err, password), password
		}
		if err != keystore.ErrDecrypt {
			// No need to prompt again if the error is not decryption-related.
			break
		}
	}
	// All trials expended to unlock account, bail out
	utils.Fatalf("Failed to unlock account %s (%v)", address, err)

	return accounts.Account{}, ""
}

func ambiguousAddrRecovery(ks *keystore.KeyStore, err *keystore.AmbiguousAddrError, auth string) accounts.Account {
	fmt.Printf("Multiple key files exist for address %x:\n", err.Addr)
	for _, a := range err.Matches {
		fmt.Println("  ", a.URL)
	}
	fmt.Println("Testing your password against all of them...")
	var match *accounts.Account
	for i, a := range err.Matches {
		if e := ks.Unlock(a, auth); e == nil {
			match = &err.Matches[i]
			break
		}
	}
	if match == nil {
		utils.Fatalf("None of the listed files could be unlocked.")
		return accounts.Account{}
	}
	fmt.Printf("Your password unlocked %s\n", match.URL)
	fmt.Println("In order to avoid this warning, you need to remove the following duplicate key files:")
	for _, a := range err.Matches {
		if a != *match {
			fmt.Println("  ", a.URL)
		}
	}
	return *match
}

// accountUpdate transitions an account from a previous format to the current
// one, also providing the possibility to change the pass-phrase.
func accountUpdate(ctx *cli.Context) error {
	if ctx.Args().Len() == 0 {
		utils.Fatalf("No accounts specified to update")
	}

	stack, err := node.NewNode(ctx.Context, &DefaultConfig)
	if err != nil {
		return err
	}

	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	for _, addr := range ctx.Args().Slice() {
		account, oldPassword := unlockAccount(ks, addr, 0, nil)
		newPassword := utils.GetPassPhraseWithList("Please give a new password. Do not forget this password.", true, 0, nil)
		if err := ks.Update(account, oldPassword, newPassword); err != nil {
			utils.Fatalf("Could not update the account: %v", err)
		}
	}
	return nil
}

func accountImport(ctx *cli.Context) error {
	if ctx.Args().Len() != 1 {
		utils.Fatalf("keyfile must be given as the only argument")
	}
	keyfile := ctx.Args().First()
	key, err := crypto.LoadECDSA(keyfile)
	if err != nil {
		utils.Fatalf("Failed to load the private key: %v", err)
	}

	stack, err := node.NewNode(ctx.Context, &DefaultConfig)
	if err != nil {
		return err
	}

	passphrase := utils.GetPassPhraseWithList("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, MakePasswordList(ctx))

	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	acct, err := ks.ImportECDSA(key, passphrase)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: {%x}\n", acct.Address)
	return nil
}
