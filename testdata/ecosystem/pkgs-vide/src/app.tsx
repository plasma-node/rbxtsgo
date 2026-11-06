import Vide, { Case, For, Provider, Show, Switch, cleanup, create, derive, effect, mount, source, spring } from "@rbxts/vide";
import { useEventListener, useInterval, useMouse } from "@rbxts/pretty-vide-utils";
import { Players, RunService } from "@rbxts/services";

function Counter(props: { readonly start: number; readonly children?: Vide.Node }) {
	const count = source(props.start);
	const doubled = derive(() => count() * 2);
	const [smooth] = spring(count, 0.5);
	effect(() => print("count", count()));
	cleanup(() => print("destroyed"));
	return (
		<frame Size={() => new UDim2(0, smooth() * 10, 0, 20)}>
			<textbutton
				Text={() => `count: ${count()} (${doubled()})`}
				Activated={() => count(count() + 1)}
				TextChanged={(text) => print(text)}
				action={(button) => print(button.Name)}
			/>
			{props.children}
		</frame>
	);
}

export function App() {
	const items = source(["a", "b", "c"]);
	const mode = source<"x" | "y">("x");
	const visible = source(true);
	useEventListener(RunService.Heartbeat, (dt) => print(dt));
	useInterval(() => mode(mode() === "x" ? "y" : "x"), 1);
	const mouse = useMouse();
	return (
		<>
			<Counter start={1}>
				<textlabel Text="child" />
			</Counter>
			<Show when={visible}>{() => <textlabel Text="shown" />}</Show>
			<Switch condition={mode}>
				<Case match="x">{() => <textlabel Text="X" />}</Case>
				<Case match="y">{() => <textlabel Text="Y" />}</Case>
			</Switch>
			<For each={items}>{(item: string, index: () => number) => <textlabel Text={() => `${index()}:${item}`} />}</For>
			{create("TextLabel")({ Text: "created" })}
			<textlabel Text={() => tostring(mouse())} />
		</>
	);
}

mount(() => <App />, Players.LocalPlayer.WaitForChild("PlayerGui"));
