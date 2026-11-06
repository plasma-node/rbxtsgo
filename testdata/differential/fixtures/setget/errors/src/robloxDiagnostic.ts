class C {
	public get v(): number {
		return 1;
	}
}


const c = new C();


print(c.v);


function f() {
	return null; //! roblox-ts
}
print(f());
