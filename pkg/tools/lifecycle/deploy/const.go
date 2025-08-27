package deploy

const toolName = "ig_deploy"

// actions for the lifecycle tool
const (
	actionDeployIG   = "deploy"
	actionUndeployIG = "undeploy"
	actionUpgradeIG  = "upgrade"
	actionIsDeployed = "is_deployed"
)

const (
	defaultChartUrl    = "oci://ghcr.io/inspektor-gadget/inspektor-gadget/charts/gadget"
	defaultReleaseUrl  = "https://api.github.com/repos/inspektor-gadget/inspektor-gadget/releases/latest"
	defaultReleaseName = "gadget"
	defaultNamespace   = "gadget"
)

var actions = []string{
	actionDeployIG,
	actionUndeployIG,
	actionUpgradeIG,
	actionIsDeployed,
}
