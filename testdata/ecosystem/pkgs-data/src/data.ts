import ProfileStore from "@rbxts/profile-store";
import ProfileService from "@rbxts/profileservice";
import { createCollection, setConfig } from "@rbxts/lapis";
import { Players } from "@rbxts/services";

interface PlayerData {
	coins: number;
	items: string[];
}
const template: PlayerData = { coins: 0, items: [] };

const store = ProfileStore.New("PlayerStore", template);
const legacy = ProfileService.GetProfileStore("Legacy", template);

export function load(player: Player) {
	const profile = store.StartSessionAsync(`Player_${player.UserId}`, { Cancel: () => player.Parent !== Players });
	if (profile === undefined) return player.Kick("no profile");
	profile.AddUserId(player.UserId);
	profile.Reconcile();
	profile.OnSessionEnd.Connect(() => player.Kick("session ended"));
	profile.Data.coins += 1;
	profile.Data.items.push("sword");
	const old = legacy.LoadProfileAsync(`Legacy_${player.UserId}`, "ForceLoad");
	old?.ListenToRelease(() => player.Kick());
	old?.Release();
	profile.EndSession();
}

setConfig({ saveAttempts: 3 });
const collection = createCollection("Players", { defaultData: template });

export async function lapis(player: Player) {
	const document = await collection.load(`${player.UserId}`);
	document.write({ ...document.read(), coins: 5 });
	await document.close();
}
