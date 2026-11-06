class Left {
    public __getshared(): number {
        return 1;
    }
}
const partial: Partial<Left> = {};
print(partial.__getshared()); //! TS2722
