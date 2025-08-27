package gadgets

const toolName = "ig_gadgets"

// actions for the gadget lifecycle tool
const (
	actionGetResults  = "get_results"
	actionStopGadget  = "stop_gadget"
	actionListGadgets = "list_running_gadgets"
)

var gadgetActions = []string{
	actionGetResults,
	actionStopGadget,
	actionListGadgets,
}
