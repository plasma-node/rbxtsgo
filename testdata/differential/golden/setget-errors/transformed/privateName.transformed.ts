class C {
    __get#hidden() {
        return 1;
    }
    public read() {
        return this.__get#hidden(); //! TS1005
    }
}
print(new C().read());
