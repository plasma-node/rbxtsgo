interface HasX {
    x: number;
}
// The accessor becomes a method, so the class doesn't have x any more.
class C implements HasX {
    public __getx() {
        return 1;
    }
}
print(new C());
