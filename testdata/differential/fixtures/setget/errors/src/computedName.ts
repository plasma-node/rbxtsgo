const key = "k";

class C {
	public get [key]() { //! TS1435 TS7008
		return 1;
	}
}

print(new C().k); //! TS2551
