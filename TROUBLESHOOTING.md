# Troubleshooting

## Common Issues

### Tool Limits
- Use manual gadget discovery: `-gadget-images=trace_dns:latest`
- In VS Code, create a [custom chat session](https://code.visualstudio.com/docs/copilot/chat/chat-modes#_custom-chat-modes) with specific tools

### Kubeconfig Issues
- Verify file exists: `~/.kube/config`
- Check Docker mount path in configuration
- Test connectivity: `kubectl cluster-info`
- For certificate paths: mount additional volumes or run binary directly

### Token Limit Reached
- Reduce scope: specify namespace, limit results (e.g., top 5)
- Use shorter timeouts
- Start a new VS Code Copilot Chat session

### Connection Problems
- Ensure VS Code MCP extension is enabled
- Verify Docker is running: `docker --version`
- Check VS Code developer console for errors
- Restart VS Code or the MCP server

### Gadget Discovery Issues
- Check internet connectivity
- Try manual specification: `-gadget-images=trace_dns:latest`
- Verify deployment: `kubectl get pods -n gadget`
- Enable debug logs: add `-log-level=debug` flag
