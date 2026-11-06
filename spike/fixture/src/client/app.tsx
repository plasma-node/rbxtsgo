import React, { useState } from "@rbxts/react";
import { createPortal, createRoot } from "@rbxts/react-roblox";
import { Players } from "@rbxts/services";

interface LabelProps {
	text: string;
	size?: UDim2;
}

function Label({ text, size }: LabelProps) {
	return <textlabel Text={text} Size={size ?? new UDim2(1, 0, 0, 20)} BackgroundTransparency={1} />;
}

function App() {
	const [count, setCount] = useState(0);
	return (
		<>
			<Label text={`count ${count}`} />
			<textbutton
				Text="more"
				Event={{
					Activated: () => setCount(count + 1),
				}}
			/>
		</>
	);
}

const playerGui = Players.LocalPlayer.WaitForChild("PlayerGui");
const root = createRoot(new Instance("Folder"));
root.render(createPortal(<App />, playerGui));
