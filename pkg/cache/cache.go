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

package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-service/api"
)

type gadgets map[string]*api.GadgetInfo

var ErrVersionedCacheNotFound = fmt.Errorf("no cache found for the specified version")

func getCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".cache", "ig-mcp-server")
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o755)
	}
	return dir, err
}

func SaveCache(version string, env string, infos map[string]*api.GadgetInfo) error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return fmt.Errorf("getting cache dir: %w", err)
	}
	cacheFile := filepath.Join(cacheDir, "cache-"+env+".json")
	file, err := os.Create(cacheFile)
	if err != nil {
		return fmt.Errorf("creating cache file: %w", err)
	}
	defer file.Close()

	data := map[string]gadgets{
		version: infos,
	}
	err = json.NewEncoder(file).Encode(data)
	if err != nil {
		return fmt.Errorf("encoding cache file: %w", err)
	}
	return nil
}

func LoadCache(version string, env string) (map[string]*api.GadgetInfo, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, fmt.Errorf("getting cache dir: %w", err)
	}
	cacheFile := filepath.Join(cacheDir, "cache-"+env+".json")
	file, err := os.Open(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("opening cache file: %w", err)
	}
	defer file.Close()

	var data map[string]gadgets
	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("decoding cache file: %w", err)
	}

	if infos, ok := data[version]; ok {
		return infos, nil
	}

	return nil, ErrVersionedCacheNotFound
}
