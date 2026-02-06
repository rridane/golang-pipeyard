// SPDX-License-Identifier: MIT
// Package: llbgraph
// File: llb_messages.go
//
// Simplified but faithful Go representation of BuildKit LLB messages.
// Inspired by https://github.com/moby/buildkit/blob/master/solver/pb/ops.proto

package llbgraph

// -----------------------------------------------------------------------------
// Node types
// -----------------------------------------------------------------------------

type LLBNodeType int

const (
	NodeTypeSource LLBNodeType = iota
	NodeTypeExec
	NodeTypeFile
)

// -----------------------------------------------------------------------------
// Core graph structures
// -----------------------------------------------------------------------------

// Input represents an incoming edge in the DAG.
type Input struct {
	Digest string // reference to another Op (dependency)
	Index  int64  // index of the input (for ordering)
}

// OpMetadata carries metadata like cache hints or frontend info.
type OpMetadata struct {
	CacheMap map[string]string
	Desc     string
	Frontend string
}

// Op represents a node in the BuildKit LLB DAG.
type Op struct {
	Inputs   []Input
	Mounts   []Mount
	OpType   LLBNodeType
	Source   *SourceOp
	Exec     *ExecOp
	File     *FileOp
	Metadata *OpMetadata
}

// -----------------------------------------------------------------------------
// SourceOp — represents a source of data (local, git, HTTP, etc.)
// -----------------------------------------------------------------------------

type SourceOp struct {
	Identifier string            // e.g. "local://context", "git://repo"
	Attrs      map[string]string // ex: exclude/include patterns, checksum hints
}

// -----------------------------------------------------------------------------
// ExecOp — represents a command executed in a container
// -----------------------------------------------------------------------------

type ExecOp struct {
	Meta     *Meta        // command metadata
	Mounts   []Mount      // input/output mounts
	Network  NetworkMode  // network isolation mode
	Security SecurityMode // security context
	Cwd      string       // working directory
	Args     []string     // command arguments
	Env      []string     // environment variables
	User     string       // user (e.g., "root")
}

// -----------------------------------------------------------------------------
// FileOp — represents filesystem mutations
// -----------------------------------------------------------------------------

type FileOp struct {
	Actions []FileAction
}

// FileAction can perform mkdir, mkfile, rm, copy, mknod, or symlink
type FileAction struct {
	Input       *Input
	Secondary   *Input
	OutputIndex int64

	Mkdir   *FileActionMkdir
	Mkfile  *FileActionMkfile
	Rm      *FileActionRm
	Copy    *FileActionCopy
	Mknod   *FileActionMknod
	Symlink *FileActionSymlink
}

// Individual file operations

type FileActionMkdir struct {
	Path string
	Mode uint32
}

type FileActionMkfile struct {
	Path string
	Data []byte
	Mode uint32
}

type FileActionRm struct {
	Path string
}

type FileActionCopy struct {
	SourcePath string
	DestPath   string
	Mode       uint32
}

type FileActionMknod struct {
	Path string
	Mode uint32
	Dev  int64
}

type FileActionSymlink struct {
	Target string
	Path   string
}

// -----------------------------------------------------------------------------
// Mounts
// -----------------------------------------------------------------------------

type Mount struct {
	Input       *Input // source input providing the content
	Selector    string // subpath in input
	Dest        string // target path inside the container
	OutputIndex int64  // output slot index
	Readonly    bool
	MountType   string // "bind", "cache", "tmpfs", "secret", "ssh", etc.
	CacheOpt    *CacheOpt
	TmpfsOpt    *TmpfsOpt
	SecretOpt   *SecretOpt
	SSHOpt      *SSHOpt
}

// CacheOpt represents cache-specific mount options
type CacheOpt struct {
	ID      string
	Sharing string // e.g. "shared", "private", "locked"
}

// TmpfsOpt represents tmpfs mount options
type TmpfsOpt struct {
	Size int64
}

// SecretOpt represents secret mount options
type SecretOpt struct {
	ID       string
	Optional bool
	UID      int
	GID      int
	Mode     uint32
}

// SSHOpt represents SSH socket mount options
type SSHOpt struct {
	ID       string
	Target   string
	Optional bool
}

// -----------------------------------------------------------------------------
// apply metadata and modes
// -----------------------------------------------------------------------------

type Meta struct {
	Args []string
	Env  []string
	Cwd  string
	User string
}

type NetworkMode string

const (
	NetworkModeNone    NetworkMode = "none"
	NetworkModeHost    NetworkMode = "host"
	NetworkModeDefault NetworkMode = "default"
)

type SecurityMode string

const (
	SecurityModeSandbox  SecurityMode = "sandbox"
	SecurityModeInsecure SecurityMode = "insecure"
)

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// NewSourceOp creates a new SourceOp node
func NewSourceOp(identifier string, attrs map[string]string) *Op {
	return &Op{
		OpType: NodeTypeSource,
		Source: &SourceOp{
			Identifier: identifier,
			Attrs:      attrs,
		},
	}
}

// NewExecOp creates a new ExecOp node
func NewExecOp(meta *Meta, mounts []Mount) *Op {
	return &Op{
		OpType: NodeTypeExec,
		Exec: &ExecOp{
			Meta:   meta,
			Mounts: mounts,
		},
	}
}

// NewFileOp creates a new FileOp node
func NewFileOp(actions []FileAction) *Op {
	return &Op{
		OpType: NodeTypeFile,
		File: &FileOp{
			Actions: actions,
		},
	}
}
