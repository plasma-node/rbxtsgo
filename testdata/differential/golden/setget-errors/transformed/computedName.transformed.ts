const key = "k";
class C {
    public __get[key]() {
        return 1;
    }
}
print(new C().__getk()); //! TS2551
