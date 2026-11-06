class C {
	get #hidden() { //! TS1435 TS7008
		return 1;
	}
	public read() {
		return this.#hidden; //! TS1005
	}
}

print(new C().read());
