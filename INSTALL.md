# Installation Guide

## Prerequisites

- A valid `kubeconfig` file with access to your Kubernetes cluster
- Docker (if using the Docker-based installation) or a compatible binary for your platform

## Kubernetes (In-Cluster)

Deploy the IG MCP Server directly into your Kubernetes cluster for use as a remote MCP endpoint (HTTP transport).

### 1. Deploy Inspektor Gadget

Use one of the following methods:

**Helm:**

```bash
IG_VERSION=$(curl -s https://api.github.com/repos/inspektor-gadget/inspektor-gadget/releases/latest | jq -r '.tag_name' | sed 's/^v//')
helm install gadget --namespace=gadget --create-namespace oci://ghcr.io/inspektor-gadget/inspektor-gadget/charts/gadget --version=$IG_VERSION
```

**kubectl:**

```bash
IG_VERSION=$(curl -s https://api.github.com/repos/inspektor-gadget/inspektor-gadget/releases/latest | jq -r '.tag_name')
kubectl apply -f https://github.com/inspektor-gadget/inspektor-gadget/releases/download/${IG_VERSION}/inspektor-gadget-${IG_VERSION}.yaml
```

### 2. Deploy the IG MCP Server

```bash
kubectl apply -f https://raw.githubusercontent.com/inspektor-gadget/ig-mcp-server/main/manifests/ig-mcp-server-all.yaml
```

### 3. Port-forward to your local machine

```bash
kubectl port-forward svc/ig-mcp-server 8080:8080 -n gadget
```

Then use `http://localhost:8080/mcp` as a remote MCP URL (transport: http) in your agent.

For example, to use it with **GitHub Copilot CLI**, add the following to `~/.copilot/mcp-config.json`:

```json
{
  "mcpServers": {
    "ig-mcp-server": {
      "tools": [
        "*"
      ],
      "type": "http",
      "url": "http://localhost:8080/mcp",
      "headers": {}
    }
  }
}
```

> **Security Note**: The IG MCP server does not currently implement built-in authentication. For production deployments, it is strongly recommended to place the server behind a reverse proxy (such as nginx, Traefik, or an API gateway) with proper authentication and TLS termination. Additionally, ensure that the service is not directly exposed to the internet and use network policies to restrict access to trusted sources only.

## Docker

<summary>
  <details>
    <summary>Install Inspektor Gadget MCP Server - Artifact Hub Discovery</summary>
    <pre><code>code --add-mcp '{
  "name": "inspektor-gadget",
  "command": "docker",
  "args": [
    "run",
    "-i",
    "--rm",
    "--volume",
    "ig-mcp-cache:/root/.cache/ig-mcp-server",
    "--mount",
    "type=bind,src=${env:HOME}/.kube/config,dst=/kubeconfig",
    "ghcr.io/inspektor-gadget/ig-mcp-server:latest",
    "-gadget-discoverer=artifacthub"
  ]
}'</code></pre>
  </details>
<details>
    <summary>Install Inspektor Gadget MCP Server - Specific Gadgets</summary>
    <pre><code>code --add-mcp '{
  "name": "inspektor-gadget",
  "command": "docker",
  "args": [
    "run",
    "-i",
    "--rm",
    "--volume",
    "ig-mcp-cache:/root/.cache/ig-mcp-server",
    "--mount",
    "type=bind,src=${env:HOME}/.kube/config,dst=/kubeconfig",
    "ghcr.io/inspektor-gadget/ig-mcp-server:latest",
    "-gadget-images=trace_dns:latest,trace_tcp:latest,snapshot_process:latest,snapshot_socket:latest"
  ]
}'</code></pre>
  </details>
</summary>

## Binary

You can head to the [Releases](https://github.com/inspektor-gadget/ig-mcp-server/releases) page and download the latest binary for your platform:

<summary>
  <details>
    <summary>Linux</summary>
    <pre><code>MCP_VERSION=$(curl -s https://api.github.com/repos/inspektor-gadget/ig-mcp-server/releases/latest | jq -r .tag_name)
MCP_ARCH=amd64
curl -sL https://github.com/inspektor-gadget/ig-mcp-server/releases/download/${MCP_VERSION}/ig-mcp-server-linux-${MCP_ARCH}.tar.gz | sudo tar -C /usr/local/bin -xzf - ig-mcp-server
</code></pre>
  </details>
  <details>
    <summary>macOS</summary>
    <pre><code>MCP_VERSION=$(curl -s https://api.github.com/repos/inspektor-gadget/ig-mcp-server/releases/latest | jq -r .tag_name)
MCP_ARCH=arm64
curl -sL https://github.com/inspektor-gadget/ig-mcp-server/releases/download/${MCP_VERSION}/ig-mcp-server-darwin-${MCP_ARCH}.tar.gz | sudo tar -C /usr/local/bin -xzf - ig-mcp-server
</code></pre>
  </details>
  <details>
    <summary>Windows</summary>
    <pre><code>$MCP_VERSION = (curl.exe -s https://api.github.com/repos/inspektor-gadget/ig-mcp-server/releases/latest | ConvertFrom-Json).tag_name
$MCP_ARCH = "amd64"
curl.exe -L "https://github.com/inspektor-gadget/ig-mcp-server/releases/download/$MCP_VERSION/ig-mcp-server-windows-$MCP_ARCH.tar.gz" -o "ig-mcp-server.tar.gz"
$destPath = "C:\Program Files\ig-mcp-server"
if (-Not (Test-Path $destPath -PathType Container)) { mkdir $destPath}
tar.exe -xzf "ig-mcp-server.tar.gz" -C "$destPath"
rm ig-mcp-server.tar.gz
Write-Host "✅ Extracted to $destPath"
Write-Host "👉 Please add '$destPath' to your PATH environment variable manually."
</code></pre>
  </details>
</summary>

After downloading, you can run the following command to add it to your VS Code MCP configuration:

<summary>
  <details>
    <summary>Install Inspektor Gadget MCP Server - Artifact Hub Discovery</summary>
    <pre><code>code --add-mcp '{
  "name": "inspektor-gadget",
  "command": "ig-mcp-server",
  "args": [
    "-gadget-discoverer=artifacthub"
  ]
}'</code></pre>
  </details>
<details>
    <summary>Install Inspektor Gadget MCP Server - Specific Gadgets</summary>
    <pre><code>code --add-mcp '{
    "name": "inspektor-gadget",
    "command": "ig-mcp-server",
    "args": [
      "-gadget-images=trace_dns:latest,trace_tcp:latest"
    ]
}'</code></pre>
    </details>
</summary>

## VS Code Setup

[VS Code supports MCP](https://code.visualstudio.com/docs/copilot/chat/mcp-servers) for connecting to the Inspektor Gadget MCP server.

### Option 1: User Settings (Global)

1. Open command palette (Ctrl+Shift+P)
2. Select "Preferences: Open User Settings (JSON)"
3. Add:

```json
{
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
}
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
