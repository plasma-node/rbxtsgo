import Roact from "@rbxts/roact";
import { Model } from "models/model";

interface Props {
	model: Model;
}

export default function Widget(props: Props) {
	return <frame Key="x" Size={new UDim2(0, props.model.sides, 0, 1)} />;
}
export const element = <textlabel Text="y" />;
