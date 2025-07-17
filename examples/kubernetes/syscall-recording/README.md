# Syscall recording using Inspektor Gadget MCP Server

## Description
This example demonstrates how to use Inspektor Gadget MCP server to record and get detailed insight of a pod.

## Environment
- Kubernetes Cluster: [AKS](https://azure.microsoft.com/en-us/products/kubernetes-service)
- Model: [Claude Sonnet 4](https://www.anthropic.com/claude/sonnet)
- MCP Client: [VS Code](https://code.visualstudio.com/docs/copilot/chat/mcp-servers)
- MCP Server: [Inspektor Gadget MCP Server](https://github.com/inspektor-gadget/ig-mcp-server)
- MCP Tools:
    - Inspektor Gadget MCP server: `is_inspektor_gadget_deployed`, `traceloop`, `snapshot_process`

### mcp.json

```json
{
  "servers": {
    "inspektor-gadget": {
      "type": "stdio",
      "command": "ig-mcp-server",
      "args": [
        "-gadget-images=traceloop,snapshot_process",
        "-read-only"
      ]
    }
  }
}
```

## Protocol

![initial prompt](media/initial-prompt.png)

> **💡 Commentary:** We start by asking about workload information in the cluster. Since we aren't using the Kubernetes MCP server, we'll use the Inspektor Gadget MCP server to get the information.

![snapshot-kube-system](media/snapshot-kube-system.png)

> **💡 Commentary:** We use the `snapshot_process` gadget to get information about the `kube-system` namespace. The gadget will return a snapshot of all processes running in the namespace with Kubernetes information.

![snapshot-response](media/snapshot-response.png)

![truncated](media/truncated.png)

> **💡 Commentary:** We chose kube-system to get a lot of information to see how the server works. It was able to fetch most of the processes running in the namespace, but also informed us that some processes were truncated due to the size of the response.

![snapshot-filter](media/snapshot-filter.png)

> **💡 Commentary:** We can ask it to filter the already fetched information to see how it handles it. Of course, this is for demonstration purposes—you can always be more specific in your prompts to manage the content size.

![snaphost-complete](media/snapshot-complete.png)

> **💡 Commentary:** Great! It's able to combine information from both runs and give us a complete picture. The AI is basically playing detective, piecing together clues from multiple sources.

![traceloop-coredns](media/traceloop-coredns.png)

> **💡 Commentary:** Now we ask it to use the `traceloop` gadget to get information about the `coredns` pod in the `kube-system` namespace. See how we specify to pick one of the pods in the namespace—because targeting is everything in observability.

![traceloop-coredns](media/traceloop-coredns-response1.png)

> **💡 Commentary:** It comes back with a very detailed response about the coredns pod. This is where things get interesting—we're essentially getting a play-by-play of what CoreDNS is doing under the hood.

![traceloop-coredns](media/traceloop-coredns-response3.png)

> **💡 Commentary:** It also provides some performance recommendations based on the data it collected. Whether these are accurate insights or the AI is just making educated guesses is hard to say, but feel free to ask follow-up questions to get better insights. Trust, but verify—especially when dealing with AI recommendations.

## Conclusion

This example demonstrates how to use the Inspektor Gadget MCP server for detailed syscall recording and process analysis in Kubernetes clusters. By combining snapshots and traceloop data, you can gain deep insights into how your workloads behave at the system level.

Give it a try the next time you need to debug mysterious performance issues, `CrashLoopBackOff` or just want to satisfy your curiosity about what your pods are really doing when nobody's watching!
