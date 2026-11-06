import { Players, RunService } from "@rbxts/services";
import * as Services from "@rbxts/services";

print(Players.LocalPlayer.Name, RunService.IsServer(), Services.Workspace.Name);
