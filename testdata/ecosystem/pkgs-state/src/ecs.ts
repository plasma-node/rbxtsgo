import * as jecs from "@rbxts/jecs";
import { Loop, World, component, useDeltaTime, useEvent, useThrottle } from "@rbxts/matter";
import { RunService } from "@rbxts/services";

const world = jecs.world();
const Position = world.component<Vector3>();
const Velocity = world.component<Vector3>();
const Dead = world.entity();
const ChildOf = world.entity();

export function jecsDemo() {
	const e = world.entity();
	world.set(e, Position, new Vector3(1, 2, 3));
	world.set(e, Velocity, Vector3.zero);
	world.add(e, Dead);
	for (const [id, position, velocity] of world.query(Position, Velocity).without(Dead)) {
		world.set(id, Position, position.add(velocity));
	}
	world.delete(e);
	return world.target(e, ChildOf);
}

const Health = component<{ value: number }>("Health");
const matterWorld = new World();

function damage(w: World) {
	for (const [id, health] of w.query(Health)) {
		if (useThrottle(1)) w.insert(id, Health({ value: health.value - useDeltaTime() }));
	}
	for (const [, player] of useEvent(RunService, "Heartbeat")) print(player);
}

const loop = new Loop(matterWorld);
loop.scheduleSystem(damage);
loop.begin({ default: RunService.Heartbeat });
