import React, { createContext, useContext, useState } from "@rbxts/react";

export interface Theme {
	readonly background: Color3;
	readonly text: Color3;
}

export const ThemeContext = createContext<Theme>({
	background: new Color3(0, 0, 0),
	text: new Color3(1, 1, 1),
});

export function ThemeProvider(props: React.PropsWithChildren<{ readonly dark?: boolean }>) {
	const [theme] = useState<Theme>(() =>
		props.dark ? { background: Color3.fromRGB(10, 10, 10), text: Color3.fromRGB(240, 240, 240) } : { background: Color3.fromRGB(240, 240, 240), text: Color3.fromRGB(10, 10, 10) },
	);
	return <ThemeContext.Provider value={theme}>{props.children}</ThemeContext.Provider>;
}

export function useTheme() {
	return useContext(ThemeContext);
}
