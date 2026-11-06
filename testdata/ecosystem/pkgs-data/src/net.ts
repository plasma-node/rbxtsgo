import Net from "@rbxts/net";
import { Context } from "@rbxts/gamejoy";
import { Zone } from "@rbxts/zone-plus";
import { Icon } from "@rbxts/topbar-plus";
import Squash from "@rbxts/squash";
import { Workspace, ReplicatedStorage } from "@rbxts/services";

export const Remotes = Net.Definitions.Create({
	Greet: Net.Definitions.ServerToClientEvent<[message: string]>(),
	Ping: Net.Definitions.ClientToServerEvent<[count: number]>(),
	Fetch: Net.Definitions.ServerAsyncFunction<(id: number) => string>(),
	Group: Net.Definitions.Namespace({
		Update: Net.Definitions.BidirectionalEvent<[value: number], [ok: boolean]>(),
	}),
});

export function server() {
	const greet = Remotes.Server.Get("Greet");
	greet.SendToAllPlayers("hello");
	Remotes.Server.Get("Ping").Connect((player, count) => print(player.Name, count));
	Remotes.Server.Get("Fetch").SetCallback((player, id) => `${player.Name}:${id}`);
	Remotes.Server.GetNamespace("Group").Get("Update");
}

export async function client() {
	Remotes.Client.Get("Greet").Connect((message) => print(message));
	Remotes.Client.Get("Ping").SendToServer(1);
	const result = await Remotes.Client.Get("Fetch").CallServerAsync(5);
	return result;
}

export function misc() {
	const context = new Context({ ActionGhosting: 0 });
	const zone = new Zone(Workspace);
	zone.localPlayerEntered.Connect(() => print("entered"));
	new Icon().setLabel("Menu").bindEvent("selected", () => print("selected"));
	const u8 = Squash.uint(1);
	const cursor = Squash.cursor();
	return { context, zone, u8, cursor, root: ReplicatedStorage };
}
