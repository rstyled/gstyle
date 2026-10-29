int straight(int x) { return x; }

int nested(int x, bool a, bool b) {
    if (a) {
        if (b) x = x + 1;
    }
    return x;
}

int alternatives(int x) {
    if (x > 2) return 2;
    else if (x > 1) return 1;
    else return 0;
}

bool logical(bool a, bool b, bool c) { return a && b && c; }
bool mixed(bool a, bool b, bool c, bool d) { return a && (b || c) && d; }

int selection(int x) {
    switch (x) {
        case 1: return 1;
        case 2: return 2;
        default: return 0;
    }
}

int cycle(int x) { if (x > 0) return cycle(x - 1); return x; }
int mutual_b(int x);
int mutual_a(int x) { if (x > 0) return mutual_b(x - 1); return x; }
int mutual_b(int x) { if (x > 0) return mutual_a(x - 1); return x; }

int chain(int x) {
    int a = x + 1;
    int b = a + 2;
    int c = b + 3;
    int d = c + 4;
    return d;
}

int caught(int x) { try { if (x) throw x; } catch (...) { return -1; } return x; }
int conditional(int x) { return x ? 1 : 0; }
int looping(int x) { while (x > 0) { if (x == 2) break; --x; } return x; }
int closure(int x) { auto fn = [x]() { if (x) return 1; return 0; }; return fn(); }

int shadowed(int x) { int value = x; { int value = 1; value = 2; } return value; }
int braced(int x) { if (x) return 1; else { if (x > 1) return 2; } return 0; }
bool negated(bool a, bool b, bool c) { return a && !(b && c); }
int predicate(int x) { if (x ? 1 : 0) return 1; return 0; }
