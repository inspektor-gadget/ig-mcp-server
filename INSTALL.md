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

| Option | Description | Default                    |
|--------|-------------|----------------------------|
| `-gadget-discoverer` | Gadget discovery method (artifacthub) | -                          |
| `-gadget-images` | Manual gadget list (e.g., 'trace_dns:latest,trace_open:latest') | -                          |
| `-artifacthub-official` | Use only official Artifact Hub gadgets | true                       |
| `-environment` | Target environment | kubernetes                 |
| `-transport` | Transport protocol | stdio, sse, streamable-http | stdio |
| `-log-level` | Logging level (debug, info, warn, error) | info                       |

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


