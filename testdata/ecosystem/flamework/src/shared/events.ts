import { Networking } from "@flamework/networking";

interface ServerEvents {
	ping(count: number): void;
}
interface ClientEvents {
	pong(count: number): void;
}
interface ServerFunctions {
	getName(id: number): string;
}
interface ClientFunctions {}

export const GlobalEvents = Networking.createEvent<ServerEvents, ClientEvents>();
export const GlobalFunctions = Networking.createFunction<ServerFunctions, ClientFunctions>();
