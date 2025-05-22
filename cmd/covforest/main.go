// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Covforest is a program for managing and analyzing multiple coverage trees
// from different machines, repositories, and timelines.
//
// Usage:
//
//	covforest <command> [arguments]
//
// The commands are:
//
//	add		add a coverage tree to the forest
//	list		list all coverage trees in the forest
//	summary		show summary statistics across all trees
//	serve		start HTTP server for exploring the forest
//	prune		remove old or invalid coverage trees
//	sync		synchronize trees from remote sources
//	help		show help for a command
//
// Use "covforest help <command>" for more information about a command.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	log.SetPrefix("covforest: ")
	log.SetFlags(0)

	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	if args[0] == "help" {
		help(args[1:])
		return
	}

	for _, cmd := range commands {
		if cmd.Name == args[0] {
			cmd.Flag.Usage = func() { cmd.Usage() }
			cmd.Flag.Parse(args[1:])
			ctx := context.Background()
			if err := cmd.Run(ctx, cmd.Flag.Args()); err != nil {
				log.Fatal(err)
			}
			return
		}
	}

	fmt.Fprintf(os.Stderr, "covforest: unknown subcommand %q\nRun 'covforest help' for usage.\n", args[0])
	os.Exit(2)
}

func usage() {
	fmt.Fprintf(os.Stderr, `Covforest is a program for managing and analyzing multiple coverage trees.

Usage:

	covforest <command> [arguments]

The commands are:

`)
	for _, cmd := range commands {
		fmt.Fprintf(os.Stderr, "\t%s\t\t%s\n", cmd.Name, cmd.Short)
	}
	fmt.Fprintf(os.Stderr, `
Use "covforest help <command>" for more information about a command.

A coverage forest manages multiple coverage trees from different sources:
- Different machines (CI/CD workers, developer machines)
- Different repositories (monorepo vs separate repos)  
- Different timelines (historical data, continuous monitoring)

Each tree maintains metadata about its source, allowing for rich
analysis and comparison across your entire development ecosystem.
`)
}

func help(args []string) {
	if len(args) == 0 {
		usage()
		return
	}
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: covforest help command\n\nToo many arguments given.\n")
		os.Exit(2)
	}

	arg := args[0]
	for _, cmd := range commands {
		if cmd.Name == arg {
			cmd.Usage()
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Unknown help topic %#q. Run 'covforest help'.\n", arg)
	os.Exit(2)
}

// A Command is an implementation of a covforest command
type Command struct {
	// Run runs the command.
	// The args are the arguments after the command name.
	Run func(ctx context.Context, args []string) error

	// UsageLine is the one-line usage message.
	UsageLine string

	// Short is the short description shown in the 'covforest help' output.
	Short string

	// Long is the long message shown in the 'covforest help <this-command>' output.
	Long string

	// Flag is a set of flags specific to this command.
	Flag flag.FlagSet

	// Name is the command name.
	Name string
}

// Usage prints the usage message for the command to stderr.
func (c *Command) Usage() {
	fmt.Fprintf(os.Stderr, "usage: %s\n", c.UsageLine)
	if c.Long != "" {
		fmt.Fprintf(os.Stderr, "%s\n", strings.TrimSpace(c.Long))
	}
}

// Runnable reports whether the command can be run; otherwise
// it is a documentation pseudo-command such as importpath.
func (c *Command) Runnable() bool {
	return c.Run != nil
}

var commands = []*Command{
	cmdAdd,
	cmdList,
	cmdSummary,
	cmdServe,
	cmdPrune,
	cmdSync,
}

func init() {
	// Set the command names from UsageLine
	for _, cmd := range commands {
		name := cmd.UsageLine
		if i := strings.Index(name, " "); i >= 0 {
			name = name[i+1:]
			if j := strings.Index(name, " "); j >= 0 {
				name = name[:j]
			}
		}
		cmd.Name = name
	}
}
