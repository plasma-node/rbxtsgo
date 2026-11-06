import { Components } from "@flamework/components";
import { BaseComponent, Component } from "@flamework/components";
import { Controller, Dependency, Flamework, OnInit, OnStart, OnTick, Reflect, Service } from "@flamework/core";
import { Players } from "@rbxts/services";
import { GlobalEvents, GlobalFunctions } from "../shared/events";

@Service({ loadOrder: 1 })
export class DataService implements OnInit, OnStart {
	private readonly names = new Map<number, string>();

	onInit() {
		print("init");
	}

	onStart() {
		const events = GlobalEvents.createServer({});
		events.ping.connect((player, count) => print(player.Name, count));
		GlobalFunctions.createServer({}).getName.setCallback((player, id) => this.names.get(id) ?? player.Name);
	}

	register(id: number, name: string) {
		this.names.set(id, name);
	}
}

@Service()
export class GameService implements OnStart, OnTick {
	constructor(private readonly data: DataService) {}
	onStart() {
		this.data.register(1, "one");
		for (const player of Players.GetPlayers()) this.data.register(player.UserId, player.Name);
	}
	onTick(dt: number) {
		print(dt);
	}
}

interface Attributes {
	speed: number;
}

@Component({ tag: "Spinner" })
export class Spinner extends BaseComponent<Attributes, Part> implements OnStart {
	onStart() {
		this.attributes.speed = 5;
		print(Flamework.id<Spinner>());
	}
}

const isConfig = Flamework.createGuard<{ readonly speed: number }>();
print(isConfig({ speed: 1 }), Reflect.getMetadata(DataService, "flamework:id"));
Flamework.addPaths("src/server");
Flamework.addPaths("src/shared");
Flamework.ignite();
const components = Dependency<Components>();
print(components, Dependency<DataService>());

export const controller = Controller;
