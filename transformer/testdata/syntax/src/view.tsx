declare namespace React {
	function createElement(tag: unknown, props?: unknown, ...children: Array<unknown>): unknown;
	const Fragment: unknown;
}
declare namespace JSX {
	interface IntrinsicElements {
		frame: { Size?: number; Key?: string; children?: unknown };
		textlabel: { Text?: string };
	}
}
const attrs = { Size: 1 };
export const element = (
	<frame {...attrs} Key="k">
		text with   spaces
		{"value"}
		<textlabel Text={`t${1}`} />
		<>
			<textlabel />
		</>
		&amp; entity &#65;
	</frame>
);
