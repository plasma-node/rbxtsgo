export type UserId = number;

export interface Profile {
	id: UserId;
	coins: number;
	tag?: string;
}

export const DEFAULT_COINS = 100;
