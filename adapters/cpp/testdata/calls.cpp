#define CPP_INC(value) ((value) + 1)

int increment(int value) { return CPP_INC(value); }

int recursive_closure(int value) {
    auto run = [value]() {
        return value ? recursive_closure(value - 1) : 0;
    };
    return run();
}

int dispatch_unknown(int (*run)(int), int value) { return run(value); }
