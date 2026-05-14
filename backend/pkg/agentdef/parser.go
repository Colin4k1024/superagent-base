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

package agentdef

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

const (
	supportedAPIVersion = "superagent/v1"
	supportedKind       = "Agent"
)

// validAgentTypes lists the accepted values for AgentSpec.Type.
var validAgentTypes = map[string]struct{}{
	"chat_model_agent": {},
	"deep_agent":       {},
	"workflow":         {},
	"supervisor":       {},
	"sequential":       {},
	"parallel":         {},
	"plan_execute":     {},
	"eino_graph":       {}, // native Eino graph from pkg/graphs registry
}

// orchestrationTypes is the subset of validAgentTypes that represent
// multi-agent coordination modes.  These types do not require a primary
// model when they delegate entirely to sub-agents, but they must have at
// least one sub_agent ref.
var orchestrationTypes = map[string]struct{}{
	"supervisor":   {},
	"sequential":   {},
	"parallel":     {},
	"plan_execute": {},
}

// namePattern validates that metadata.name consists only of lowercase
// alphanumerics and hyphens (e.g. "research-agent", "my-bot-v2").
var namePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// Parse parses raw YAML bytes into an AgentDefinition and validates the result.
// Returns an error if the document is malformed or fails validation.
func Parse(data []byte) (*AgentDefinition, error) {
	var def AgentDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("agentdef: parse: %w", err)
	}
	if err := Validate(&def); err != nil {
		return nil, err
	}
	return &def, nil
}

// ParseFile reads the file at path and delegates to Parse.
func ParseFile(path string) (*AgentDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("agentdef: parse file %q: %w", path, err)
	}
	def, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("agentdef: parse file %q: %w", path, err)
	}
	return def, nil
}

// Validate checks that an AgentDefinition contains all required fields and
// that their values are within accepted bounds.
func Validate(def *AgentDefinition) error {
	if def.APIVersion != supportedAPIVersion {
		return fmt.Errorf("agentdef: unsupported apiVersion %q (expected %q)", def.APIVersion, supportedAPIVersion)
	}
	if def.Kind != supportedKind {
		return fmt.Errorf("agentdef: unsupported kind %q (expected %q)", def.Kind, supportedKind)
	}
	if def.Metadata.Name == "" {
		return fmt.Errorf("agentdef: metadata.name is required")
	}
	if !namePattern.MatchString(def.Metadata.Name) {
		return fmt.Errorf("agentdef: metadata.name %q must match [a-z0-9-]+", def.Metadata.Name)
	}
	if _, ok := validAgentTypes[def.Spec.Type]; !ok {
		return fmt.Errorf("agentdef: spec.type %q must be one of: chat_model_agent, deep_agent, workflow, supervisor, sequential, parallel, plan_execute", def.Spec.Type)
	}
	_, isOrchestration := orchestrationTypes[def.Spec.Type]
	if isOrchestration {
		// Orchestration types require at least one sub_agent reference.
		if len(def.Spec.SubAgents) == 0 {
			return fmt.Errorf("agentdef: spec.type %q requires at least one sub_agents entry", def.Spec.Type)
		}
		for i, sa := range def.Spec.SubAgents {
			if sa.Ref == "" {
				return fmt.Errorf("agentdef: sub_agents[%d].ref is required", i)
			}
		}
	} else if def.Spec.Type == "workflow" {
		// Workflow type requires a non-nil workflow spec with at least one node
		// and one edge.
		if def.Spec.Workflow == nil {
			return fmt.Errorf("agentdef: spec.type %q requires spec.workflow to be set", def.Spec.Type)
		}
		if len(def.Spec.Workflow.Nodes) == 0 {
			return fmt.Errorf("agentdef: spec.workflow must have at least one node")
		}
		if len(def.Spec.Workflow.Edges) == 0 {
			return fmt.Errorf("agentdef: spec.workflow must have at least one edge")
		}
	} else {
		// Non-orchestration types require a primary model.
		if def.Spec.Model.Primary == "" {
			return fmt.Errorf("agentdef: spec.model.primary is required")
		}
	}
	return nil
}
