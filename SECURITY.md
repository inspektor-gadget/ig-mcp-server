# Security Guide

You can limit the permissions of the Inspektor Gadget MCP Server by creating a dedicated service account with restricted access. This is useful when you want to use Inspektor Gadget MCP server without granting full cluster admin permissions.

## Quick Setup

### 1. Apply the Manifest

We start by creating a service account and role binding assuming you have already [deployed Inspektor Gadget](https://inspektor-gadget.io/docs/latest/reference/install-kubernetes) in `gadget` namespace. This service account will have limited permissions to interact with the Kubernetes API.

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ig-mcp-server-sa
  namespace: gadget
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: gadget
  name: ig-mcp-server-role
rules:
  - apiGroups: [ "" ]
    resources: [ "pods/portforward" ]
    verbs: [ "create" ]
  - apiGroups: [ "" ]
    resources: [ "pods"]
    verbs: [ "list" ]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: ig-mcp-server-binding
  namespace: gadget
subjects:
  - kind: ServiceAccount
    name: ig-mcp-server-sa
    namespace: gadget
roleRef:
  kind: Role
  name: ig-mcp-server-role
  apiGroup: rbac.authorization.k8s.io
EOF
```

### 2. Extract Token

Extract the token and set it in the kubeconfig file.

```bash
TOKEN=$(kubectl create token ig-mcp-server-sa --namespace gadget)
kubectl config set-credentials ig-mcp-server-sa --token=${TOKEN}
```

the token has an expiration time, so you may need to regenerate it periodically.

### 3. Use with Inspektor Gadget MCP Server

Use the `--user` to specify the service account when running the MCP server. This ensures that the server operates with the permissions granted to the `ig-mcp-server` service account:

```bash
{
  "servers": {
    "inspektor-gadget": {
      "type": "stdio",
      "command": "ig-mcp-server",
      "args": [
        "-gadget-discoverer=artifacthub",
        "-user=ig-mcp-server-sa",
        "-read-only"
      ]
    }
  }
}
```

you can also use `--token` to specify the token directly or use `--kubeconfig` to point to a specific kubeconfig file that contains the service account credentials.

## Conclusion

This setup allows you to run the Inspektor Gadget MCP server with limited permissions, enhancing security while still providing the necessary functionality for monitoring and troubleshooting Kubernetes clusters.
Also, management tools like `deploy_inspektor_gadget` and `undeploy_inspektor_gadget` won't work with this setup, as they require cluster admin permissions to deploy/undeploy Inspektor Gadget.
