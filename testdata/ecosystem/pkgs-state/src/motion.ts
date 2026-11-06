import * as Flipper from "@rbxts/flipper";
import spr from "@rbxts/spr";
import Fusion from "@rbxts/fusion";
import Rodux from "@rbxts/rodux";

const motor = new Flipper.SingleMotor(0);
motor.onStep((value) => print(value));
motor.setGoal(new Flipper.Spring(1, { frequency: 4, dampingRatio: 1 }));

const group = new Flipper.GroupMotor({ x: 0, y: 0 });
group.setGoal({ x: new Flipper.Linear(1, { velocity: 2 }), y: new Flipper.Instant(5) });

export function animate(label: TextLabel) {
	spr.target(label, 0.8, 4, { Position: new UDim2(0.5, 0, 0.5, 0), BackgroundTransparency: 0.5 });
	spr.stop(label);
}

const Value = Fusion.Value;
const count = Value(0);
export const view = Fusion.New("TextLabel")({
	Text: Fusion.Computed(() => `n = ${count.get()}`),
	[Fusion.OnEvent("MouseEnter")]: () => count.set(count.get() + 1),
});

type Action = { type: "add"; amount: number };
const reducer = Rodux.createReducer<number, Action>(0, {
	add: (state: number, action: Action) => state + action.amount,
});
export const store = new Rodux.Store(reducer, undefined, [Rodux.loggerMiddleware]);
store.dispatch({ type: "add", amount: 1 });
