import Maid from "@rbxts/maid";
import { Janitor } from "@rbxts/janitor";
import { Dumpster } from "@rbxts/dumpster";
import Signal from "@rbxts/signal";
import LemonSignal, { Connection } from "@rbxts/lemon-signal";
import Make from "@rbxts/make";
import { Workspace, RunService, Players } from "@rbxts/services";

export class Widget {
	private readonly maid = new Maid();
	private readonly janitor = new Janitor<{ Tween: Tween }>();
	private readonly dumpster = new Dumpster();
	readonly changed = new Signal<(value: number) => void>();
	readonly ready = new LemonSignal<[string, number]>();
	private connection?: Connection;

	constructor(readonly part: Part) {
		this.maid.GiveTask(part);
		this.maid.GiveTask(RunService.Heartbeat.Connect((dt) => print(dt)));
		this.maid["extra"] = () => print("cleaned");
		this.janitor.Add(part, "Destroy");
		this.janitor.Add(this.changed, "Destroy");
		this.janitor.Add(() => print("janitor"), true);
		this.dumpster.dump(part).dump(() => print("dumped")).dump(this.ready, (s) => s.Destroy());
		this.connection = this.ready.Connect((name, n) => print(name, n));
		this.changed.Connect((v) => print(v));
	}

	update(v: number) {
		this.changed.Fire(v);
		this.ready.Fire("update", v);
		const [value] = this.changed.Wait();
		return value;
	}

	destroy() {
		this.connection?.Disconnect();
		this.maid.DoCleaning();
		this.janitor.Cleanup();
		this.dumpster.burn();
	}
}

export function build() {
	const frame = Make("Frame", {
		Name: "Root",
		Size: new UDim2(1, 0, 1, 0),
		Children: [
			Make("TextLabel", { Name: "Title", Text: "hello" }),
			Make("TextButton", { Name: "Button", Activated: () => print("click") }),
		],
		Parent: Players.LocalPlayer.WaitForChild("PlayerGui"),
	});
	const part = Make("Part", { Name: "P", Anchored: true, Parent: Workspace });
	return new Widget(part).update(frame.Size.X.Offset);
}
