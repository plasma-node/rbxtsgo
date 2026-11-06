import React, { StrictMode, Suspense, lazy } from "@rbxts/react";
import { createPortal, createRoot } from "@rbxts/react-roblox";
import { Players } from "@rbxts/services";
import { ClassComponent, Counter, FancyButton, List, MemoLabel, Pure, Spread, Wrapper } from "./components";
import { ThemeProvider } from "./context";

const playerGui = Players.LocalPlayer.WaitForChild("PlayerGui");
const root = createRoot(new Instance("Folder"));

function App() {
	return (
		<StrictMode>
			<ThemeProvider dark>
				<screengui ResetOnSpawn={false}>
					<Counter start={3} />
					<List items={["a", "b", "c"]} />
					<Spread extra={{ text: "x" }} />
					<Wrapper>
						<MemoLabel text="memo" />
						<ClassComponent name="cc" />
						<Pure v={1} />
					</Wrapper>
					{createPortal(<textlabel Text="portal" />, playerGui)}
				</screengui>
			</ThemeProvider>
		</StrictMode>
	);
}

root.render(<App />);
export = FancyButton;
