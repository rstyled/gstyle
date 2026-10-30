extern void consume(int value);
extern void may_throw();

int guard_candidate(bool ok, int value) {
    if (ok) {
        consume(value);
        consume(value + 1);
        return value;
    } else {
        return 0;
    }
}

int small_branch(bool ok, int value) {
    if (ok) { consume(value); return value; }
    else { return 0; }
}

int effectful_else(bool ok, int value) {
    if (ok) { consume(value); consume(value + 1); return value; }
    else { consume(0); return 0; }
}

bool same_explicit(bool ok) {
    if (ok) { return true; } else { return true; }
}

bool same_tail(bool ok) {
    if (ok) { return false; }
    return false;
}

bool different_results(bool ok) {
    if (ok) { return true; } else { return false; }
}

bool branch_effect(bool ok) {
    if (ok) { consume(1); return true; } else { return true; }
}

int non_boolean(bool ok) {
    if (ok) { return 1; } else { return 1; }
}

template <bool Enabled> bool compile_time_branch() {
    if constexpr (Enabled) { return true; } else { return true; }
}

bool condition_binding() {
    if (int value = 1; value > 0) { return true; } else { return true; }
}

void empty_catch() {
    try { may_throw(); } catch (...) {}
}

void typed_empty_catch() {
    try { may_throw(); } catch (int value) {}
}

void handled_catch() {
    try { may_throw(); } catch (...) { consume(1); }
}

void rethrow_catch() {
    try { may_throw(); } catch (...) { throw; }
}

void documented_catch() {
    try { may_throw(); } catch (...) { /* Failure is optional here. */ }
}

int dependency_cycle(int value) {
    int a = value;
    int b = value + 1;
    int c = value + 2;
    while (value-- > 0) {
        a = b + c;
        b = a + c;
        c = a + b;
    }
    return a + b + c;
}

int dispatch_ladder(bool first, bool second, int value) {
    if (first) { return value; }
    else if (second) { consume(value); consume(value + 1); return value; }
    else { return 0; }
}
