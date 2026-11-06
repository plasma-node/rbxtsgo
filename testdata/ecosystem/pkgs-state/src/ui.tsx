import React, { useEffect } from "@rbxts/react";
import { ReflexProvider, UseProducerHook, UseSelectorHook, useProducer, useSelector } from "@rbxts/react-reflex";
import { createMotion, createSpring, config } from "@rbxts/ripple";
import { RootState, root, selectDone } from "./reflex";

const useRootProducer: UseProducerHook<typeof root> = useProducer;
const useRootSelector: UseSelectorHook<typeof root> = useSelector;

function TodoList() {
	const producer = useRootProducer();
	const todos = useRootSelector((state: RootState) => state.todos.todos);
	const done = useRootSelector(selectDone);
	const motion = createMotion(0, { start: true });
	useEffect(() => {
		motion.spring(1, config.gentle);
		return motion.onChange((value) => print(value));
	}, []);
	return (
		<frame>
			{todos.map((todo) => (
				<textbutton key={todo.id} Text={todo.text} Event={{ Activated: () => producer.toggle(todo.id) }} />
			))}
			<textlabel Text={`${done.size()} done`} />
		</frame>
	);
}

export function App() {
	const spring = createSpring(new Vector3(0, 0, 0), { frequency: 2 });
	spring.setGoal(new Vector3(1, 2, 3));
	return (
		<ReflexProvider producer={root}>
			<TodoList />
		</ReflexProvider>
	);
}
