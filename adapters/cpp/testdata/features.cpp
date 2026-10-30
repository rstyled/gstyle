struct Customer {
    int id;
    int age;
    bool active;
    int read() const { return age; }
};

struct Account { int id; int balance; };
struct Record { int a; int b; int c; int d; int e; int f; };
struct Contact { int age; int region; };

int customer_age(Customer &customer) { return customer.age; }
int account_balance(Account &account) { return account.balance; }
int account_amount(Account &account) { return account.balance + 1; }
int account_credit(Account &account) { return account.balance + 2; }

void change(Customer &customer, Account &account) {
    customer.id = 7;
    account.balance = 10;
}

bool same_customer(Customer &left, Customer &right) {
    return left.age == right.age && left.active == right.active;
}

bool same_contact(Contact &left, Contact &right) {
    return left.age == right.age && left.region == right.region;
}

bool eligible(Customer &customer) { return customer.active && customer.age > 18; }
bool may_buy(Customer &customer) { return customer.active && customer.age > 18; }

Record record_one(int value) { if (value < 0) value = 0; return Record{value, 2, 3, 4, 5, 6}; }
Record record_two(int value) { if (value < 0) value = 0; return Record{value, 8, 9, 4, 5, 6}; }

int address_one(int city, int zip, int amount) { return city + zip + amount; }
int address_two(int city, int zip, int amount) { return city + zip + amount + 1; }
int address_three(int city, int zip, int amount) { return city + zip + amount + 2; }

extern int sqlite3_exec(int);
void persist(int value) { sqlite3_exec(value); }

int pure_helper(int value) { return value + 1; }
int pure_caller(int value) { return pure_helper(value); }
extern void opaque(int value);
void opaque_caller(int value) { opaque(value); }
void change_caller(Customer &customer, Account &account) { change(customer, account); }

enum class State { New, Ready, Busy, Done, Failed };
int state_value(State state) {
    switch (state) {
    case State::New: return 0;
    case State::Ready: return 1;
    case State::Busy: return 2;
    case State::Done: return 3;
    case State::Failed: return 4;
    }
    return -1;
}

struct Globals { static int current; volatile int flag; };
int Globals::current = 0;
int read_global(Globals &value) { return value.current; }
void write_global(Globals &value) { value.current = 1; }
int read_volatile(Globals &value) { return value.flag; }

struct TestMutex {};
struct TestGuard { explicit TestGuard(TestMutex &mutex); };
TestGuard tiny_wrapper(TestMutex &mutex) { return TestGuard{mutex}; }
int arithmetic(int value) { return (value + 1) * (value - 1) + value / 2; }
int with_call(int value) { return pure_helper(value + 1) * (value - 1) + value / 2; }
