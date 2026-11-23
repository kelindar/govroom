#include <cstdlib>
#include <iostream>
#include <sstream>

#include "utils/input_parser.h"
#include "utils/output_json.h"

namespace vroom {

// The host discards this WASI instance after every solve.
[[noreturn]] void exit_with_error(const Exception& error) {
  io::write_to_json(error);
  std::exit(error.error_code);
}

} // namespace vroom

int main() {
  std::ostringstream request;
  request << std::cin.rdbuf();

  vroom::Input input;
  vroom::io::parse(input, request.str(), false);
  const auto solution = input.solve(vroom::DEFAULT_EXPLORATION_LEVEL, 1);
  vroom::io::write_to_json(solution, "", input.report_distances());
}
