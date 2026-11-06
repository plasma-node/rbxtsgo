import replecs from "@rbxts/replecs";
import * as jecs from "@rbxts/jecs";
import { world as createWorld } from "@rbxts/jecs/src/jecs";
import "@rbxts/jest-extended";
import { Players } from "@rbxts/services";

const world = createWorld();
const server = replecs.create_server(world);
const client = replecs.create_client(world);
const Position = world.component<Vector3>();
world.add(Position, replecs.shared);
server.init(world);
client.init(world);

export function entity() {
	const e = world.entity();
	world.set(e, Position, Vector3.zero);
	world.add(e, jecs.Wildcard);
	return Players.GetPlayers().size() + e;
}
