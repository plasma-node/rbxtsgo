export interface Shape {
	sides: number;
}
export class Square implements Shape {
	public sides = 4;
}
export const makeSquare = () => new Square();
