/*
   Copyright 2014 CoreOS, Inc.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"text/tabwriter"

	"github.com/coreos/locksmith/etcd"
	"github.com/coreos/locksmith/consul"
	"github.com/coreos/locksmith/lock"
	"github.com/coreos/locksmith/version"
)

const (
	cliName        = "locksmithctl"
	cliDescription = `Manage the cluster wide reboot lock.`
)

var (
	out *tabwriter.Writer

	commands      []*Command
	globalFlagset *flag.FlagSet = flag.NewFlagSet("locksmithctl", flag.ExitOnError)

	globalFlags = struct {
		Backend      string
		Debug        bool
		EtcdEndpoint string
		EtcdKeyFile  string
		EtcdCertFile string
		EtcdCAFile   string
		ConsulEndpoint string
		Version      bool
	}{}
)

func init() {
	out = new(tabwriter.Writer)
	out.Init(os.Stdout, 0, 8, 1, '\t', 0)

	globalFlagset.BoolVar(&globalFlags.Debug, "debug", false, "Print out debug information to stderr.")
	globalFlagset.StringVar(&globalFlags.EtcdEndpoint, "etc-endpoint", "http://127.0.0.1:4001", "etcd endpoint for locksmith. Defaults to the local instance.")
	globalFlagset.StringVar(&globalFlags.EtcdKeyFile, "etcd-keyfile", "", "etcd key file authentication")
	globalFlagset.StringVar(&globalFlags.EtcdCertFile, "etcd-certfile", "", "etcd cert file authentication")
	globalFlagset.StringVar(&globalFlags.EtcdCAFile, "etcd-cafile", "", "etcd CA file authentication")
	globalFlagset.StringVar(&globalFlags.ConsulEndpoint, "consul-endpoint", "127.0.0.1:8500", "consul endpoint for locksmith. Defaults to the local instance.")
	globalFlagset.StringVar(&globalFlags.Backend, "backend", "etcd", "etcd or consul as backend for locksmith. Defaults to the etcd.")
	globalFlagset.BoolVar(&globalFlags.Version, "version", false, "Print the version and exit.")

	commands = []*Command{
		cmdHelp,
		cmdLock,
		cmdReboot,
		cmdSendNeedReboot,
		cmdSetMax,
		cmdStatus,
		cmdUnlock,
	}
}

type Command struct {
	Name        string                  // Name of the Command and the string to use to invoke it
	Summary     string                  // One-sentence summary of what the Command does
	Usage       string                  // Usage options/arguments
	Description string                  // Detailed description of command
	Flags       flag.FlagSet            // Set of flags associated with this command
	Run         func(args []string) int // Run a command with the given arguments, return exit status
}

func getAllFlags() (flags []*flag.Flag) {
	return getFlags(globalFlagset)
}

func getFlags(flagset *flag.FlagSet) (flags []*flag.Flag) {
	flags = make([]*flag.Flag, 0)
	flagset.VisitAll(func(f *flag.Flag) {
		flags = append(flags, f)
	})
	return
}

func main() {
	globalFlagset.Parse(os.Args[1:])
	var args = globalFlagset.Args()

	progName := path.Base(os.Args[0])

	if globalFlags.Version {
		fmt.Printf("%s version %s\n", progName, version.Version)
		os.Exit(0)
	}

	if progName == "locksmithd" {
		flagsFromEnv("LOCKSMITHD", globalFlagset)
		os.Exit(runDaemon())
	}

	// no command specified - trigger help
	if len(args) < 1 {
		args = append(args, "help")
	}

	flagsFromEnv("LOCKSMITHCTL", globalFlagset)

	var cmd *Command

	// determine which Command should be run
	for _, c := range commands {
		if c.Name == args[0] {
			cmd = c
			if err := c.Flags.Parse(args[1:]); err != nil {
				fmt.Println(err.Error())
				os.Exit(2)
			}
			break
		}
	}

	if cmd == nil {
		fmt.Printf("%v: unknown subcommand: %q\n", cliName, args[0])
		fmt.Printf("Run '%v help' for usage.\n", cliName)
		os.Exit(2)
	}

	os.Exit(cmd.Run(cmd.Flags.Args()))
}

// getLockClient returns an initialized EtcdLockClient, using an etcd
// client configured from the global etcd flags
func getClient() (lock.LockClient, error) {
    if globalFlags.Backend == "etcd" {
        return getClientEtcd()
    } else if globalFlags.Backend == "consul" {
        return getClientConsul()
    } else {
        fmt.Printf("Unsupported backend %s\n",  globalFlags.Backend)
        os.Exit(2)
        return nil, nil
    }
}
func getClientConsul() (lock.LockClient, error) {
	ec, err := consul.NewClient(globalFlags.ConsulEndpoint)
	if err != nil {
		return nil, err
	}
	lc, err := lock.NewConsulKVLockClient(ec)
	if err != nil {
		return nil, err
	}
	return lc, err
}
func getClientEtcd() (lock.LockClient, error) {
	var ti *etcd.TLSInfo
	if globalFlags.EtcdCAFile != "" || globalFlags.EtcdCertFile != "" || globalFlags.EtcdKeyFile != "" {
		ti = &etcd.TLSInfo{
			CertFile: globalFlags.EtcdCertFile,
			KeyFile:  globalFlags.EtcdKeyFile,
			CAFile:   globalFlags.EtcdCAFile,
		}
	}
	ec, err := etcd.NewClient([]string{globalFlags.EtcdEndpoint}, ti)
	if err != nil {
		return nil, err
	}
	lc, err := lock.NewEtcdLockClient(ec)
	if err != nil {
		return nil, err
	}
	return lc, err
}

// flagsFromEnv parses all registered flags in the given flagset,
// and if they are not already set it attempts to set their values from
// environment variables. Environment variables take the name of the flag but
// are UPPERCASE, have the given prefix, and any dashes are replaced by
// underscores - for example: some-flag => PREFIX_SOME_FLAG
func flagsFromEnv(prefix string, fs *flag.FlagSet) {
	alreadySet := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		alreadySet[f.Name] = true
	})
	fs.VisitAll(func(f *flag.Flag) {
		if !alreadySet[f.Name] {
			key := strings.ToUpper(prefix + "_" + strings.Replace(f.Name, "-", "_", -1))
			val := os.Getenv(key)
			if val != "" {
				fs.Set(f.Name, val)
			}
		}
	})
}
