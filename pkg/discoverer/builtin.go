// Copyright 2025 The Inspektor Gadget authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package discoverer

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

const SourceBuiltin = "builtin"

//go:embed data/gadgets.json
var embeddedGadgetsData []byte

type BuiltinGadgetPackages struct {
	Packages []BuiltinGadgetPackage `json:"packages"`
}

type BuiltinGadgetPackage struct {
	NormalizedName string `json:"normalized_name"`
	Description    string `json:"description"`
	ContainerImage string `json:"container_image"`
}

type builtinDiscoverer struct{}

func NewBuiltinDiscoverer() Discoverer {
	return &builtinDiscoverer{}
}

func (d *builtinDiscoverer) ListGadgets() ([]Gadget, error) {
	log.Debug("Loading gadgets from embedded data")

	// Parse the embedded JSON
	var packages BuiltinGadgetPackages
	if err := json.Unmarshal(embeddedGadgetsData, &packages); err != nil {
		return nil, fmt.Errorf("failed to parse embedded gadgets JSON: %w", err)
	}

	// Convert to Gadget structs
	gadgets := make([]Gadget, 0, len(packages.Packages))
	for _, pkg := range packages.Packages {
		if pkg.ContainerImage == "" {
			log.Warn("Skipping gadget with empty container image", "name", pkg.NormalizedName)
			continue
		}

		gadgets = append(gadgets, Gadget{
			Image:       pkg.ContainerImage,
			Description: pkg.Description,
		})
	}

	log.Debug("Loaded gadgets from embedded data", "count", len(gadgets))
	return gadgets, nil
}
