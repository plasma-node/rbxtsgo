interface LabelProps {
	readonly text: string;
	readonly order?: number;
	readonly children?: ReactNode;
}

function Label({ text, order = 0 }: LabelProps) {
	return <textlabel Text={text} LayoutOrder={order} />;
}

function Counter(props: { readonly start: number }) {
	const [count, setCount] = React.useState(props.start);
	const [visible, setVisible] = React.useState(true);
	return (
		<frame Size={new UDim2(1, 0, 1, 0)}>
			<Label key="count" text={`count: ${count}`} order={1} />
			{visible && <Label key="visible" text="visible" />}
			{!visible ? <Label key="hidden" text="hidden" /> : undefined}
			<textbutton Text="inc" Event={{ Activated: () => setCount(count + 1) }} />
			<textbutton Text="toggle" Event={{ Activated: () => setVisible((v) => !v) }} />
		</frame>
	);
}

function List(props: { readonly items: readonly string[] }) {
	return (
		<>
			{props.items.map((item, index) => (
				<Label key={`item-${index}`} text={item} order={index} />
			))}
		</>
	);
}

function Spread(props: { readonly extra: Partial<LabelProps> }) {
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

function Wrapper(props: { readonly children?: ReactNode }) {
	return <frame>{props.children}</frame>;
}

export const tree = (
	<Wrapper>
		<Counter start={3} />
		<List items={["a", "b"]} />
		<Spread extra={{ text: "x" }} />
		<Label text="plain">text child</Label>
		{[<Label key="a" text="a" />, <Label key="b" text="b" />]}
	</Wrapper>
);
