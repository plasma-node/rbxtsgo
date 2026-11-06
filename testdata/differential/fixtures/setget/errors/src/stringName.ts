class C {
	public get "a b"() { //! TS1435 TS7008
		return 1;
	}
}

print(new C()["a b"]);
