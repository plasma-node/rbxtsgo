// A stand-in for @rbxts/react, so this fixture needs no package. Only the shapes that JSX needs.
interface ReactElement {
	readonly _nominal_ReactElement: unique symbol;
}
type ReactNode = ReactElement | string | number | boolean | undefined | ReadonlyArray<ReactNode>;
type FunctionComponent<P> = (props: P) => ReactElement | undefined;

declare const React: {
	createElement(type: unknown, props?: unknown, ...children: unknown[]): ReactElement;
	readonly Fragment: unique symbol;
	readonly Event: { readonly Activated: unique symbol };
	useState<T>(initial: T): LuaTuple<[T, (value: T | ((previous: T) => T)) => void]>;
};

declare namespace JSX {
	type Element = ReactElement;
	interface ElementChildrenAttribute {
		children: {};
	}
	interface IntrinsicAttributes {
		key?: string | number;
	}
	interface IntrinsicElements {
		frame: { Size?: UDim2; Visible?: boolean; children?: ReactNode; Event?: { Activated?: () => void } };
		textlabel: { Text?: string; Size?: UDim2; LayoutOrder?: number };
		textbutton: { Text?: string; Event?: { Activated?: () => void }; ref?: { current?: TextButton } };
	}
}
