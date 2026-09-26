#include <thread>
#include <vector>

void importantStuff() {
    std::this_thread::sleep_for(std::chrono::minutes(42));
}

int main(int argc, char* argv[]) {
    std::vector<std::thread> threads;
    for (int i = 0; i < 41; ++i) {
        threads.push_back(std::thread(importantStuff));
    }

    std::cout << "running...\n" << std::flush;
    importantStuff();

    return 0;
}
