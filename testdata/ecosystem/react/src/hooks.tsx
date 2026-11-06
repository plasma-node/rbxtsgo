import React, { useBinding, useEffect, useState } from "@rbxts/react";
import {
	useDebounceState,
	useEventListener,
	useLatest,
	useMountEffect,
	usePrevious,
	useThrottleCallback,
	useUpdateEffect,
} from "@rbxts/pretty-react-hooks";
import { RunService, UserInputService } from "@rbxts/services";

export function HookUser(props: { readonly value: number }) {
	const previous = usePrevious(props.value);
	const latest = useLatest(props.value);
	const [debounced, setDebounced] = useDebounceState(0, { wait: 0.2 });
	const [binding, setBinding] = useBinding(0);
	const throttled = useThrottleCallback((n: number) => setBinding(n), { wait: 0.1 });

	useEventListener(RunService.Heartbeat, (dt) => {
		setBinding(binding.getValue() + dt);
	});
	useEventListener(UserInputService.InputBegan, (input) => {
		if (input.KeyCode === Enum.KeyCode.E) throttled.run(latest.current);
	});
	useMountEffect(() => print("mount"));
	useUpdateEffect(() => setDebounced(props.value), [props.value]);
	useEffect(() => print(previous, debounced), [previous, debounced]);

	return <textlabel Text={binding.map((n) => `${n}`)} Size={binding.map((n) => new UDim2(0, n, 0, 10))} />;
}

export function Local() {
	const [n, setN] = useState(0);
	return <textbutton Text={`${n}`} Event={{ Activated: () => setN(n + 1) }} />;
}
