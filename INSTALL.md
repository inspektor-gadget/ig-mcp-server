# Installation Guide

## VS Code Setup

[VS Code supports MCP](https://code.visualstudio.com/docs/copilot/chat/mcp-servers) for connecting to the Inspektor Gadget MCP server.

### Option 1: User Settings (Global)

1. Open command palette (Ctrl+Shift+P)
2. Select "Preferences: Open User Settings (JSON)"
3. Add:

```json
`{
  "mcp": {
    "inspektor-gadget": {
      "type": "stdio",
      "command": "docker",
      "args": [
            "run",
            "-i",
            "--mount",
            "type=bind,src=${env:HOME}/.kube/config,dst=/kubeconfig",
            "ghcr.io/inspektor-gadget/ig-mcp-server:latest",
            "-gadget-discoverer=artifacthub"
      ]
    }
  }
}`
```

### Option 2: Workspace Settings (Project-specific)

Create `.vscode/mcp.json` in your project directory:

```json
{
  "servers": {
    "inspektor-gadget": {
      "type": "stdio",
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--mount",
        "type=bind,src=${env:HOME}/.kube/config,dst=/kubeconfig",
        "ghcr.io/inspektor-gadget/ig-mcp-server:latest",
        "-gadget-discoverer=artifacthub"
      ]
    }
  }
}
```

## Configuration Options

Key command-line options:

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `-gadget-discoverer` | Gadget discoverer to use (artifacthub) | - | One of `-gadget-discoverer` or `-gadget-images` |
| `-gadget-images` | Comma-separated list of gadget images to use (e.g. 'trace_dns:latest,trace_open:latest') | - | One of `-gadget-discoverer` or `-gadget-images` |
| `-artifacthub-official` | Use only official gadgets from Artifact Hub | true | No |
| `-environment` | Environment to use (currently only 'kubernetes' is supported) | kubernetes | No |
| `-context` | The name of the kubeconfig context to use | - | No |
| `-kubeconfig` | Path to the kubeconfig file to use | - | No |
| `-user` | The name of the kubeconfig user to use | - | No |
| `-token` | Bearer token to use for authentication | - | No |
| `-read-only` | Run the server in read-only mode | false | No |
| `-transport` | Transport to use (stdio, sse, streamable-http) | stdio | No |
| `-transport-host` | Host for the transport | localhost | No |
| `-transport-port` | Port for the transport | 8080 | No |
| `-log-level` | Log level (debug, info, warn, error) | - | No |
| `-version` | Print version and exit | - | No |

**Important**: You must specify either `-gadget-discoverer` or `-gadget-images`. The server will fail to start without one of these options.

For all options:

```bash
docker run ghcr.io/inspektor-gadget/ig-mcp-server -h
```

## Building from Source

```bash
git clone https://github.com/inspektor-gadget/ig-mcp-server.git
cd ig-mcp-server
make
```
This should build binary (`ig-mcp-server`) which can be used directly:

```json
{
  "mcp": {
    "inspektor-gadget": {
      "type": "stdio",
      "command": "ig-mcp-server",
      "args": [
        "-gadget-discoverer=artifacthub"
      ]
    }
  }
}
```

To build it for all the plaforms you can use `make ig-mcp-server-all`.
