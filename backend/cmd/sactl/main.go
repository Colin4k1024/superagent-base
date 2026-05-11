/*
 * Copyright 2025 superagent-ai Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// sactl is the Superagent control CLI.
//
// Usage:
//
//	sactl skill search <query>
//	sactl skill install <name>@<version>
//	sactl skill list
//	sactl skill uninstall <name>
//	sactl agent apply -f <yaml-file>
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/superagent-ai/superagent-base/backend/pkg/skill"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "skill":
		runSkill(os.Args[2:])
	case "agent":
		runAgent(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `sactl - Superagent control CLI

Usage:
  sactl skill search <query>
  sactl skill install <name>@<version>
  sactl skill list
  sactl skill uninstall <name>
  sactl agent apply -f <yaml-file>

Environment variables:
  SKILLSHUB_URL    Base URL for the SkillsHub service (default: http://localhost:8080)
  SKILLSHUB_TOKEN  Authentication token for the SkillsHub service`)
}

// hubClient returns a configured HubClient using environment variables.
func hubClient() skill.HubClient {
	baseURL := os.Getenv("SKILLSHUB_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	token := os.Getenv("SKILLSHUB_TOKEN")
	return skill.NewHTTPHubClient(skill.HTTPHubClientConfig{
		BaseURL:   baseURL,
		AuthToken: token,
	})
}

func runSkill(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sactl skill <search|install|list|uninstall>")
		os.Exit(1)
	}

	switch args[0] {
	case "search":
		runSkillSearch(args[1:])
	case "install":
		runSkillInstall(args[1:])
	case "list":
		runSkillList(args[1:])
	case "uninstall":
		runSkillUninstall(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown skill sub-command: %s\n", args[0])
		os.Exit(1)
	}
}

func runSkillSearch(args []string) {
	fs := flag.NewFlagSet("skill search", flag.ExitOnError)
	limit := fs.Int("limit", 20, "maximum number of results")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: sactl skill search <query>")
		os.Exit(1)
	}

	query := strings.Join(fs.Args(), " ")
	client := hubClient()
	results, err := client.Search(context.Background(), query, skill.SearchOpts{Limit: *limit})
	if err != nil {
		fmt.Fprintf(os.Stderr, "search failed: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("no skills found")
		return
	}
	fmt.Printf("%-30s %-10s %s\n", "NAME", "VERSION", "DESCRIPTION")
	for _, s := range results {
		fmt.Printf("%-30s %-10s %s\n", s.Name, s.Version, s.Description)
	}
}

func runSkillInstall(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sactl skill install <name>@<version>")
		os.Exit(1)
	}

	ref := args[0]
	name, version, found := strings.Cut(ref, "@")
	if !found || version == "" {
		fmt.Fprintln(os.Stderr, "error: specify skill as <name>@<version>")
		os.Exit(1)
	}

	client := hubClient()
	instance, err := client.Install(context.Background(), name, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "install failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("installed %s@%s (status: %s)\n", instance.Meta.Name, instance.Meta.Version, instance.Status)
}

func runSkillList(args []string) {
	_ = args // no flags yet
	client := hubClient()
	instances, err := client.List(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "list failed: %v\n", err)
		os.Exit(1)
	}

	if len(instances) == 0 {
		fmt.Println("no skills installed")
		return
	}
	fmt.Printf("%-30s %-10s %s\n", "NAME", "VERSION", "STATUS")
	for _, inst := range instances {
		fmt.Printf("%-30s %-10s %s\n", inst.Meta.Name, inst.Meta.Version, inst.Status)
	}
}

func runSkillUninstall(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sactl skill uninstall <name>")
		os.Exit(1)
	}
	name := args[0]
	client := hubClient()
	if err := client.Uninstall(context.Background(), name); err != nil {
		fmt.Fprintf(os.Stderr, "uninstall failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("uninstalled %s\n", name)
}

func runAgent(args []string) {
	fs := flag.NewFlagSet("agent apply", flag.ExitOnError)
	yamlFile := fs.String("f", "", "path to agent YAML definition file")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 || fs.Arg(0) != "apply" {
		fmt.Fprintln(os.Stderr, "usage: sactl agent apply -f <yaml-file>")
		os.Exit(1)
	}
	if *yamlFile == "" {
		fmt.Fprintln(os.Stderr, "error: -f flag is required")
		os.Exit(1)
	}

	// Placeholder: agent apply is not yet implemented.
	fmt.Printf("[placeholder] would apply agent definition from: %s\n", *yamlFile)
}
