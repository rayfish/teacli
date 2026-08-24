// Package cmd contains SSH key, GPG key and deploy key commands.
package cmd

import (
	"fmt"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var keyCmd = &cobra.Command{
	Use:     "key",
	Aliases: []string{"keys"},
	Short:   "Manage SSH, GPG and deploy keys",
}

var sshKeyCmd = &cobra.Command{
	Use:     "ssh",
	Aliases: []string{"ssh-key", "ssh-keys"},
	Short:   "Manage account SSH keys",
}

var sshKeyListCmd = &cobra.Command{
	Use:   "list [<username>]",
	Short: "List your SSH keys, or another user's public ones",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSSHKeyList,
}

var sshKeyGetCmd = &cobra.Command{
	Use:   "get <key-id>",
	Short: "Get one SSH key",
	Args:  cobra.ExactArgs(1),
	RunE:  runSSHKeyGet,
}

var sshKeyAddCmd = &cobra.Command{
	Use:     "add <title>",
	Aliases: []string{"create"},
	Short:   "Add an SSH key to your account",
	Args:    cobra.ExactArgs(1),
	RunE:    runSSHKeyAdd,
}

var sshKeyDeleteCmd = &cobra.Command{
	Use:     "delete <key-id>",
	Aliases: []string{"rm"},
	Short:   "Delete one of your SSH keys",
	Args:    cobra.ExactArgs(1),
	RunE:    runSSHKeyDelete,
}

var gpgKeyCmd = &cobra.Command{
	Use:     "gpg",
	Aliases: []string{"gpg-key", "gpg-keys"},
	Short:   "Manage account GPG keys",
}

var gpgKeyListCmd = &cobra.Command{
	Use:   "list [<username>]",
	Short: "List your GPG keys, or another user's",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runGPGKeyList,
}

var gpgKeyGetCmd = &cobra.Command{
	Use:   "get <key-id>",
	Short: "Get one GPG key",
	Args:  cobra.ExactArgs(1),
	RunE:  runGPGKeyGet,
}

var gpgKeyAddCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"create"},
	Short:   "Add a GPG key to your account",
	Args:    cobra.NoArgs,
	RunE:    runGPGKeyAdd,
}

var gpgKeyDeleteCmd = &cobra.Command{
	Use:     "delete <key-id>",
	Aliases: []string{"rm"},
	Short:   "Delete one of your GPG keys",
	Args:    cobra.ExactArgs(1),
	RunE:    runGPGKeyDelete,
}

var gpgKeyTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Get the token you must sign to verify a GPG key",
	Args:  cobra.NoArgs,
	RunE:  runGPGKeyToken,
}

var gpgKeyVerifyCmd = &cobra.Command{
	Use:   "verify <key-id>",
	Short: "Verify a GPG key with a signed token",
	Args:  cobra.ExactArgs(1),
	RunE:  runGPGKeyVerify,
}

var deployKeyCmd = &cobra.Command{
	Use:     "deploy",
	Aliases: []string{"deploy-key", "deploy-keys"},
	Short:   "Manage repository deploy keys",
}

var deployKeyListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List a repository's deploy keys",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDeployKeyList,
}

var deployKeyGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <key-id>",
	Short: "Get one deploy key",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runDeployKeyGet,
}

var deployKeyAddCmd = &cobra.Command{
	Use:     "add [<owner>/<repo>] <title>",
	Aliases: []string{"create"},
	Short:   "Add a deploy key to a repository",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runDeployKeyAdd,
}

var deployKeyDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <key-id>",
	Aliases: []string{"rm"},
	Short:   "Delete a deploy key",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runDeployKeyDelete,
}

func init() {
	RootCmd.AddCommand(keyCmd)
	keyCmd.AddCommand(sshKeyCmd, gpgKeyCmd, deployKeyCmd)
	sshKeyCmd.AddCommand(sshKeyListCmd, sshKeyGetCmd, sshKeyAddCmd, sshKeyDeleteCmd)
	gpgKeyCmd.AddCommand(gpgKeyListCmd, gpgKeyGetCmd, gpgKeyAddCmd, gpgKeyDeleteCmd,
		gpgKeyTokenCmd, gpgKeyVerifyCmd)
	deployKeyCmd.AddCommand(deployKeyListCmd, deployKeyGetCmd, deployKeyAddCmd, deployKeyDeleteCmd)

	addPageFlags(sshKeyListCmd, gpgKeyListCmd, deployKeyListCmd)

	for _, c := range []*cobra.Command{sshKeyAddCmd, deployKeyAddCmd} {
		c.Flags().StringP("key", "k", "", "The public key itself")
		c.Flags().StringP("key-file", "f", "", "Read the public key from this file, or - for stdin")
		c.Flags().Bool("read-only", false, "Grant read-only access")
	}
	gpgKeyAddCmd.Flags().StringP("key", "k", "", "The armoured public key")
	gpgKeyAddCmd.Flags().StringP("key-file", "f", "", "Read the armoured key from this file, or - for stdin")
	gpgKeyAddCmd.Flags().String("signature", "", "Armoured signature over the verification token")

	gpgKeyVerifyCmd.Flags().String("signature", "",
		"Armoured signature over the token from 'key gpg token' (required)")
	gpgKeyVerifyCmd.MarkFlagRequired("signature")
}

// publicKeyMaterial resolves a key body from --key or --key-file.
func publicKeyMaterial(cmd *cobra.Command) (string, error) {
	key, err := bodyFromFlags(cmd, "key", "key-file")
	if err != nil {
		return "", err
	}
	if key == "" {
		return "", errors.NewValidationError("key required",
			map[string]interface{}{"hint": "pass --key <text> or --key-file <path>"})
	}
	return key, nil
}

func emitPublicKeys(cmd *cobra.Command, keys []*gitea.PublicKey) error {
	printer := getPrinter(cmd)
	return printer.Emit(keys, func() error {
		if len(keys) == 0 {
			printer.Println("No keys found.")
			return nil
		}
		rows := make([][]string, 0, len(keys))
		for _, k := range keys {
			rows = append(rows, []string{
				fmt.Sprintf("%d", k.ID), k.Title, k.KeyType, k.Fingerprint,
				yesNo(k.ReadOnly), renderValue(k.Created),
			})
		}
		return printer.PrintTable(
			[]string{"ID", "Title", "Type", "Fingerprint", "Read only", "Created"}, rows)
	})
}

func runSSHKeyList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	keys, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.PublicKey, *gitea.Response, error) {
		if len(args) == 1 {
			return client.ListPublicKeys(args[0], gitea.ListPublicKeysOptions{ListOptions: lo})
		}
		return client.ListMyPublicKeys(gitea.ListPublicKeysOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitPublicKeys(cmd, keys)
}

func runSSHKeyGet(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "key id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	key, resp, err := client.GetPublicKey(id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "ssh key", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(key, func() error {
		var d detail
		d.always("ID", key.ID)
		d.add("Title", key.Title)
		d.add("Type", key.KeyType)
		d.add("Fingerprint", key.Fingerprint)
		d.always("Read only", key.ReadOnly)
		d.add("Owner", userName(key.Owner))
		d.add("Created", key.Created)
		d.add("Key", key.Key)
		return d.print(printer)
	})
}

func runSSHKeyAdd(cmd *cobra.Command, args []string) error {
	key, err := publicKeyMaterial(cmd)
	if err != nil {
		return err
	}
	readOnly, _ := cmd.Flags().GetBool("read-only")

	if dryRunf(cmd, "Would add SSH key %q", args[0]) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	created, resp, err := client.CreatePublicKey(gitea.CreateKeyOption{
		Title: args[0], Key: key, ReadOnly: readOnly,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, created, "Added SSH key %s (id %d)", created.Title, created.ID)
}

func runSSHKeyDelete(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "key id")
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would delete SSH key %d", id) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeletePublicKey(id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "ssh key", args[0])
	}
	return emitMessage(cmd, okMessage("key deleted"), "Deleted SSH key %d", id)
}

func runGPGKeyList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	keys, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.GPGKey, *gitea.Response, error) {
		if len(args) == 1 {
			return client.ListGPGKeys(args[0], gitea.ListGPGKeysOptions{ListOptions: lo})
		}
		return client.ListMyGPGKeys(&gitea.ListGPGKeysOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(keys, func() error {
		if len(keys) == 0 {
			printer.Println("No GPG keys found.")
			return nil
		}
		rows := make([][]string, 0, len(keys))
		for _, k := range keys {
			emails := make([]string, 0, len(k.Emails))
			for _, e := range k.Emails {
				if e != nil {
					emails = append(emails, e.Email)
				}
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", k.ID), k.KeyID, renderValue(emails),
				yesNo(k.CanSign), yesNo(k.Verified), renderValue(k.Created),
			})
		}
		return printer.PrintTable(
			[]string{"ID", "Key ID", "Emails", "Can sign", "Verified", "Created"}, rows)
	})
}

func runGPGKeyGet(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "key id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	key, resp, err := client.GetGPGKey(id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "gpg key", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(key, func() error {
		var d detail
		d.always("ID", key.ID)
		d.add("Key ID", key.KeyID)
		d.add("Primary key ID", key.PrimaryKeyID)
		d.always("Can sign", key.CanSign)
		d.always("Can encrypt", key.CanEncryptComms)
		d.always("Can certify", key.CanCertify)
		d.always("Verified", key.Verified)
		d.add("Created", key.Created)
		d.add("Expires", key.Expires)
		return d.print(printer)
	})
}

func runGPGKeyAdd(cmd *cobra.Command, args []string) error {
	key, err := publicKeyMaterial(cmd)
	if err != nil {
		return err
	}
	signature, _ := cmd.Flags().GetString("signature")

	if dryRunf(cmd, "Would add a GPG key") {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	created, resp, err := client.CreateGPGKey(gitea.CreateGPGKeyOption{
		ArmoredKey: key, Signature: signature,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, created, "Added GPG key %s (id %d)", created.KeyID, created.ID)
}

func runGPGKeyDelete(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "key id")
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would delete GPG key %d", id) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteGPGKey(id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "gpg key", args[0])
	}
	return emitMessage(cmd, okMessage("key deleted"), "Deleted GPG key %d", id)
}

func runGPGKeyToken(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	token, resp, err := client.GetGPGKeyVerificationToken()
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]string{"token": token}, func() error {
		printer.Println(token)
		return nil
	})
}

func runGPGKeyVerify(cmd *cobra.Command, args []string) error {
	signature, _ := cmd.Flags().GetString("signature")

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would verify GPG key %s", args[0]) {
		return nil
	}

	key, resp, err := client.VerifyGPGKey(gitea.VerifyGPGKeyOption{
		KeyID: args[0], Signature: signature,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, key, "Verified GPG key %s", key.KeyID)
}

func runDeployKeyList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	keys, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.DeployKey, *gitea.Response, error) {
		return client.ListDeployKeys(owner, repo, gitea.ListDeployKeysOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(keys, func() error {
		if len(keys) == 0 {
			printer.Println("No deploy keys found.")
			return nil
		}
		rows := make([][]string, 0, len(keys))
		for _, k := range keys {
			rows = append(rows, []string{
				fmt.Sprintf("%d", k.ID), k.Title, k.Fingerprint,
				yesNo(k.ReadOnly), renderValue(k.Created),
			})
		}
		return printer.PrintTable([]string{"ID", "Title", "Fingerprint", "Read only", "Created"}, rows)
	})
}

func runDeployKeyGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	id, err := int64Arg(args[1], "key id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	key, resp, err := client.GetDeployKey(owner, repo, id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "deploy key", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(key, func() error {
		var d detail
		d.always("ID", key.ID)
		d.add("Title", key.Title)
		d.add("Fingerprint", key.Fingerprint)
		d.always("Read only", key.ReadOnly)
		d.add("Created", key.Created)
		d.add("Key", key.Key)
		return d.print(printer)
	})
}

func runDeployKeyAdd(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	title := args[1]

	key, err := publicKeyMaterial(cmd)
	if err != nil {
		return err
	}
	readOnly, _ := cmd.Flags().GetBool("read-only")

	if dryRunf(cmd, "Would add deploy key %q to %s/%s", title, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	created, resp, err := client.CreateDeployKey(owner, repo, gitea.CreateKeyOption{
		Title: title, Key: key, ReadOnly: readOnly,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, created, "Added deploy key %s (id %d)", created.Title, created.ID)
}

func runDeployKeyDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	id, err := int64Arg(args[1], "key id")
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would delete deploy key %d from %s/%s", id, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteDeployKey(owner, repo, id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "deploy key", args[1])
	}

	return emitMessage(cmd, okMessage("key deleted"), "Deleted deploy key %d", id)
}
