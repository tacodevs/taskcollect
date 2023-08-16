#!/usr/bin/python3

from pathlib import Path
import platform
import subprocess
import sys

# TODO: Copy res files to tests/data automatically when a test is run?

package_list = ["daymap", "errors", "gclass", "logger", "plat", "server"]

def exec_test(package) -> None:
    cwd = Path.cwd().joinpath("src", package)
    print()
    print(f"Testing package: {package}")
    print(f"  located at {cwd}")
    print("----------------------------------------------------------------------")

    cmd = "go test"
    subprocess.run(
        cmd,
        cwd=cwd,
        shell=True
    )


def prep_test(packages) -> None:
    if packages == "all":
        print(f"Running `go test` on all {len(package_list)} packages")
        for p in package_list:
            exec_test(p)
        return
    
    # TODO: Implement option to test individual packages


def print_help() -> None:
    cmd = "python3 test.py"
    if platform.system().lower() == "windows":
        cmd = "py test.py"
    print(
        "--- TaskCollect: Tests help ---\n\n"
        "USAGE:\n"
        f"    {cmd} [args] [command]\n"
        "\n"
        "COMMANDS:\n"
        "    all    Test all packages\n"
        "    help   Shows this command\n"
    )

def main(argc: int, argv: list[str]):
    if argc == 1:
        print("See 'help' for more information")
        prep_test("all")
    elif argv[1] == "all":
        prep_test("all")
    elif argv[1] == "help":
        print_help()
    else:
        print("Invalid argument")

if __name__ == "__main__":
    argv = sys.argv
    argc = len(argv)
    main(argc, argv)
