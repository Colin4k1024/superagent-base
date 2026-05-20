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
//	sactl agent apply   -f <yaml-file> [--dry-run] [--output json|table]
//	sactl agent validate -f <yaml-file> [--remote]  [--output json|table]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/superagent-ai/superagent-base/backend/pkg/agentdef"
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
  sactl skill search <query>              Search the SkillsHub for skills
  sactl skill install <name>@<version>    Install a skill
  sactl skill list                        List installed skills
  sactl skill uninstall <name>            Uninstall a skill

  sactl agent apply    -f <yaml-file>     Apply an agent definition
                       [--dry-run]          Validate only, do not send to server
                       [--output json|table] Output format (default: table)

  sactl agent validate -f <yaml-file>     Validate an agent definition locally
                       [--remote]           Also validate against the server
                       [--output json|table] Output format (default: table)

Environment variables:
  SKILLSHUB_URL         Base URL for the SkillsHub service (default: http://localhost:8080)
  SKILLSHUB_TOKEN       Authentication token for the SkillsHub service
  SUPERAGENT_URL        Base URL for the Superagent server  (default: http://localhost:8888)
  SUPERAGENT_ADMIN_KEY  Bearer token for admin API endpoints`)
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
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sactl agent <apply|validate> -f <yaml-file>")
		os.Exit(1)
	}
	switch args[0] {
	case "apply":
		runAgentApply(args[1:])
	case "validate":
		runAgentValidate(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown agent subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func runAgentApply(args []string) {
	fs := flag.NewFlagSet("agent apply", flag.ExitOnError)
	yamlFile := fs.String("f", "", "path to agent YAML definition file")
	dryRun := fs.Bool("dry-run", false, "validate only, do not send to server")
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if *yamlFile == "" {
		fmt.Fprintln(os.Stderr, "error: -f flag is required")
		os.Exit(1)
	}

	data, err := os.ReadFile(*yamlFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		os.Exit(1)
	}

	// Local validation.
	errs := localValidate(data)
	if len(errs) > 0 {
		printValidateResult("", false, errs, *output)
		os.Exit(1)
	}

	// Parse once more to extract the name for the PUT path.
	def, _ := agentdef.Parse(data)

	if *dryRun {
		printApplyResult(def.Metadata.Name, string(def.Spec.Type), "dry-run", "local validation passed", *output)
		return
	}

	client := NewAdminClient()
	result, err := client.ApplyAgent(def.Metadata.Name, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "apply failed: %v\n", err)
		os.Exit(1)
	}
	agentType := string(def.Spec.Type)
	printApplyResult(result.Name, agentType, result.Status, result.Message, *output)
}

func runAgentValidate(args []string) {
	fs := flag.NewFlagSet("agent validate", flag.ExitOnError)
	yamlFile := fs.String("f", "", "path to agent YAML definition file")
	remote := fs.Bool("remote", false, "also validate against the server")
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if *yamlFile == "" {
		fmt.Fprintln(os.Stderr, "error: -f flag is required")
		os.Exit(1)
	}

	data, err := os.ReadFile(*yamlFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		os.Exit(1)
	}

	// Local validation.
	errs := localValidate(data)
	if len(errs) > 0 {
		printValidateResult(*yamlFile, false, errs, *output)
		os.Exit(1)
	}

	// Remote validation (optional).
	if *remote {
		client := NewAdminClient()
		result, err := client.ValidateAgent(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "remote validate failed: %v\n", err)
			os.Exit(1)
		}
		if !result.Valid {
			printValidateResult(*yamlFile, false, result.Errors, *output)
			os.Exit(1)
		}
		printValidateResult(*yamlFile, true, nil, *output)
		return
	}

	printValidateResult(*yamlFile, true, nil, *output)
}

// localValidate parses and validates YAML content using the agentdef package.
// Returns a slice of error strings (empty means valid).
func localValidate(yamlContent []byte) []string {
	_, err := agentdef.Parse(yamlContent)
	if err != nil {
		return []string{err.Error()}
	}
	return nil
}

// printApplyResult prints the result of an apply operation in the requested format.
func printApplyResult(name, agentType, status, message, format string) {
	if format == "json" {
		out := map[string]string{
			"name":    name,
			"type":    agentType,
			"status":  status,
			"message": message,
		}
		b, _ := json.Marshal(out)
		fmt.Println(string(b))
		return
	}
	// table
	fmt.Printf("%-30s %-20s %-10s %s\n", "NAME", "TYPE", "STATUS", "MESSAGE")
	fmt.Printf("%-30s %-20s %-10s %s\n", name, agentType, status, message)
}

// printValidateResult prints the result of a validate operation.
func printValidateResult(name string, valid bool, errs []string, format string) {
	if name == "" {
		name = "-"
	}
	validStr := "true"
	if !valid {
		validStr = "false"
	}
	errStr := strings.Join(errs, "; ")

	if format == "json" {
		out := map[string]interface{}{
			"name":   name,
			"valid":  valid,
			"errors": errs,
		}
		if errs == nil {
			out["errors"] = []string{}
		}
		b, _ := json.Marshal(out)
		fmt.Println(string(b))
		return
	}
	// table
	fmt.Printf("%-30s %-7s %s\n", "NAME", "VALID", "ERRORS")
	fmt.Printf("%-30s %-7s %s\n", name, validStr, errStr)
}
