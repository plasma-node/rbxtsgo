import { InferState, combineProducers, createProducer, createSelector, loggerMiddleware, shallowEqual } from "@rbxts/reflex";
import Immut from "@rbxts/immut";

interface TodoState {
	readonly todos: readonly { readonly id: number; readonly text: string; readonly done: boolean }[];
	readonly nextId: number;
}

const initialTodos: TodoState = { todos: [], nextId: 1 };

export const todos = createProducer(initialTodos, {
	add: (state, text: string) => ({
		todos: [...state.todos, { id: state.nextId, text, done: false }],
		nextId: state.nextId + 1,
	}),
	toggle: (state, id: number) => ({
		...state,
		todos: state.todos.map((todo) => (todo.id === id ? { ...todo, done: !todo.done } : todo)),
	}),
	clear: () => initialTodos,
});

export const settings = createProducer({ volume: 1, muted: false }, {
	setVolume: (state, volume: number) => ({ ...state, volume }),
	toggleMute: (state) => ({ ...state, muted: !state.muted }),
});

export const root = combineProducers({ todos, settings });
export type RootState = InferState<typeof root>;

export const selectDone = createSelector([(state: RootState) => state.todos.todos], (list) => list.filter((todo) => todo.done));

root.applyMiddleware(loggerMiddleware);

export function demo() {
	root.add("write tests");
	root.toggle(1);
	root.setVolume(0.5);
	const unsubscribe = root.subscribe(selectDone, shallowEqual, (done) => print(done.size()));
	const updated = Immut.produce(root.getState(), (draft) => {
		draft.settings.volume = 0;
	});
	unsubscribe();
	return updated;
}
