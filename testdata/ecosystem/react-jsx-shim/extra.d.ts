declare module "react/jsx-runtime" {
	import React = require("@rbxts/react");
	export import JSX = React.JSX;
	export const jsx: unknown;
	export const jsxs: unknown;
	export const Fragment: unknown;
}
