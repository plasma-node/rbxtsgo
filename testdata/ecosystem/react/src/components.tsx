import React, { forwardRef, memo, useCallback, useEffect, useMemo, useReducer, useRef, useState } from "@rbxts/react";
import { useTheme } from "./context";

interface LabelProps {
	readonly text: string;
	readonly size?: UDim2;
	readonly order?: number;
}

export function Label({ text, size = new UDim2(0, 100, 0, 20), order = 0 }: LabelProps) {
	const theme = useTheme();
	return <textlabel Text={text} Size={size} LayoutOrder={order} BackgroundColor3={theme.background} TextColor3={theme.text} />;
}

export const MemoLabel = memo(Label, (a, b) => a.text === b.text);

export const FancyButton = forwardRef<TextButton, { readonly label: string; readonly onClick?: () => void }>((props, ref) => {
	return (
		<textbutton
			ref={ref}
			Text={props.label}
			Size={new UDim2(0, 120, 0, 30)}
			Event={{
				Activated: () => props.onClick?.(),
				MouseEnter: () => print("enter"),
			}}
			Change={{
				Text: (rbx) => print("text changed", rbx.Text),
			}}
		/>
	);
});

type Action = { readonly type: "inc" } | { readonly type: "dec" } | { readonly type: "set"; readonly value: number };

function reducer(state: number, action: Action): number {
	switch (action.type) {
		case "inc":
			return state + 1;
		case "dec":
			return state - 1;
		case "set":
			return action.value;
	}
}

export function Counter(props: { readonly start?: number }) {
	const [count, dispatch] = useReducer(reducer, props.start ?? 0);
	const [visible, setVisible] = useState(true);
	const buttonRef = useRef<TextButton>();
	const doubled = useMemo(() => count * 2, [count]);
	const toggle = useCallback(() => setVisible((v) => !v), []);

	useEffect(() => {
		const button = buttonRef.current;
		if (button) print("mounted", button.Name);
		return () => print("cleanup", count);
	}, [count]);

	return (
		<frame Size={new UDim2(1, 0, 1, 0)} BackgroundTransparency={1}>
			<uilistlayout SortOrder={Enum.SortOrder.LayoutOrder} Padding={new UDim(0, 4)} />
			<Label key="count" text={`count: ${count} (${doubled})`} order={1} />
			{visible && <Label key="visible" text="visible" order={2} />}
			{!visible ? <Label key="hidden" text="hidden" order={2} /> : undefined}
			<FancyButton ref={buttonRef} label="inc" onClick={() => dispatch({ type: "inc" })} />
			<FancyButton label="dec" onClick={() => dispatch({ type: "dec" })} />
			<FancyButton label="toggle" onClick={toggle} />
		</frame>
	);
}

export function List(props: { readonly items: readonly string[] }) {
	return (
		<>
			{props.items.map((item, index) => (
				<Label key={`item-${index}`} text={item} order={index} />
			))}
		</>
	);
}

export function Spread(props: { readonly extra: Partial<LabelProps> }) {
	const base: LabelProps = { text: "base" };
	return (
		<>
			<Label {...base} />
			<Label {...base} {...props.extra} order={5} />
			<Label text="late" {...props.extra} />
			<frame {...{ Size: new UDim2(0, 1, 0, 1) }} />
		</>
	);
}

export function Wrapper(props: React.PropsWithChildren) {
	return <frame>{props.children}</frame>;
}

export class ClassComponent extends React.Component<{ readonly name: string }, { readonly n: number }> {
	state = { n: 0 };
	componentDidMount() {
		this.setState({ n: this.state.n + 1 });
	}
	render() {
		return <Label text={`${this.props.name}:${this.state.n}`} />;
	}
}

export class Pure extends React.PureComponent<{ readonly v: number }> {
	render() {
		return <textlabel Text={tostring(this.props.v)} />;
	}
}
