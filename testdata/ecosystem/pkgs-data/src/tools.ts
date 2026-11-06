import { CmdrClient } from "@rbxts/cmdr";
import TestEZ from "@rbxts/testez";
import { ReplicatedStorage } from "@rbxts/services";

export function setup() {
	CmdrClient.SetActivationKeys([Enum.KeyCode.F2]);
	CmdrClient.SetPlaceName("Demo");
	CmdrClient.Registry.RegisterHooksIn(ReplicatedStorage);
}

export function test(root: Instance) {
	TestEZ.run(root, (results) => print(results.failureCount));
}
