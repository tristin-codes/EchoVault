// Copyright 2024 Kelvin Clement Mwinuka
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

package internal

import (
	"context"
	"github.com/echovault/echovault/internal/clock"
	"net"
	"time"
)

type KeyData struct {
	Value    interface{}
	ExpireAt time.Time
}

type ContextServerID string
type ContextConnID string

type ApplyRequest struct {
	Type         string   `json:"Type"` // command | delete-key
	ServerID     string   `json:"ServerID"`
	ConnectionID string   `json:"ConnectionID"`
	CMD          []string `json:"CMD"`
	Key          string   `json:"Key"`
}

type ApplyResponse struct {
	Error    error
	Response []byte
}

type SnapshotObject struct {
	State                      map[string]KeyData
	LatestSnapshotMilliseconds int64
}

// KeyExtractionFuncResult is the return type of the KeyExtractionFunc for the command/subcommand.
type KeyExtractionFuncResult struct {
	Channels  []string // The pubsub channels the command accesses. For non pubsub commands, this should be an empty slice.
	ReadKeys  []string // The keys the command reads from. If no keys are read, this should be an empty slice.
	WriteKeys []string // The keys the command writes to. If no keys are written to, this should be an empty slice.
}

// KeyExtractionFunc is included with every command/subcommand. This function returns a KeyExtractionFuncResult object.
// The return value of this function is used in the ACL layer to determine whether the connection is allowed to
// execute this command.
// The cmd parameter is a string slice of the command. All the keys are extracted from this command.
type KeyExtractionFunc func(cmd []string) (KeyExtractionFuncResult, error)

// HandlerFuncParams is the object passed to a command handler when a command is triggered.
// These params are provided to commands by the EchoVault engine to help the command hook into functions from the
// echovault package.
type HandlerFuncParams struct {
	// Command is the string slice contains the command (e.g []string{"SET", "key", "value"})
	Command []string
	// Connection is the connection that triggered this command.
	// Do not write the response directly to the connection, return it from the function.
	Connection *net.Conn
	// Context is the context passed from the EchoVault instance.
	Context context.Context
	// DeleteKey deletes the specified key. Returns an error if the deletion was unsuccessful.
	DeleteKey func(key string) error
	// GetACL returns the EchoVault instance's ACL engine.
	// There's no need to use this outside of the acl package,
	// ACL authorizations for all commands will be handled automatically by the EchoVault instance as long as the
	// commands KeyExtractionFunc returns the correct keys.
	GetACL func() interface{}
	// GetAllCommands returns all the commands loaded in the EchoVault instance.
	GetAllCommands func() []Command
	// GetClock gets the clock used by the server.
	// Use this when making use of time methods like .Now and .After.
	// This inversion of control is a helper for testing as the clock is automatically mocked in tests.
	GetClock func() clock.Clock
	// GetExpiry returns the expiry time of a key.
	GetExpiry func(key string) time.Time
	// GetKeys returns all keys in the server store
	GetKeys func() []string
	// GetLatestSnapshotTime returns the latest snapshot timestamp
	GetLatestSnapshotTime func() int64
	// GetPubSub returns the EchoVault instance's PubSub engine.
	// There's no need to use this outside of the pubsub package.
	GetPubSub func() interface{}
	// GetValues retrieves the values from the specified keys.
	// Non-existent keys will be nil.
	GetValues func(ctx context.Context, keys []string) map[string]interface{}
	// KeysExist returns a map that specifies which keys exist in the keyspace.
	KeysExist func(keys []string) map[string]bool
	// ListModules returns the list of modules loaded in the EchoVault instance.
	ListModules func() []string
	// LoadModule loads the provided module with the given args passed to the module's
	// key extraction and handler functions.
	LoadModule func(path string, args ...string) error
	// RewriteAOF triggers a compaction of the commands logs by the EchoVault instance.
	RewriteAOF func() error
	// SetValues sets each of the keys with their corresponding values in the provided map.
	SetValues func(ctx context.Context, entries map[string]interface{}) error
	// Set expiry sets the expiry time of the key.
	SetExpiry func(ctx context.Context, key string, expire time.Time, touch bool)
	// TakeSnapshot triggers a snapshot by the EchoVault instance.
	TakeSnapshot func() error
	// UnloadModule removes the specified module.
	// This unloads both custom modules and internal modules.
	UnloadModule func(module string)
}

// HandlerFunc is a functions described by a command where the bulk of the command handling is done.
// This function returns a byte slice which contains a RESP2 response. The response from this function
// is forwarded directly to the client connection that triggered the command.
// In embedded mode, the response is parsed and a native Go type is returned to the caller.
type HandlerFunc func(params HandlerFuncParams) ([]byte, error)

type Command struct {
	Command     string       // The command keyword (e.g. "set", "get", "hset").
	Module      string       // The module this command belongs to. All the available modules are in the `constants` package.
	Categories  []string     // The ACL categories this command belongs to. All the available categories are in the `constants` package.
	Description string       // The description of the command. Includes the command syntax.
	SubCommands []SubCommand // The list of subcommands for this command. Empty if the command has no subcommands.
	Sync        bool         // Specifies if command should be synced across replication cluster
	KeyExtractionFunc
	HandlerFunc
}

type SubCommand struct {
	Command     string   // The keyword for this subcommand. (Check the acl module for an example of subcommands within a command).
	Module      string   // The module this subcommand belongs to. Should be the same as the parent command.
	Categories  []string // The ACL categories the subcommand belongs to.
	Description string   // The description of the subcommand. Includes syntax.
	Sync        bool     // Specifies if sub-command should be synced across replication cluster
	KeyExtractionFunc
	HandlerFunc
}
