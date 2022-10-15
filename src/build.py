#!/usr/bin/python3

import os
#import platform
import subprocess
import sys


platforms = {
    "darwin": ["amd64", "arm64"],
    "linux": ["386", "amd64", "arm", "arm64"],
    "windows": ["386", "amd64"]
}

def build(op_sys: str, arch: str):
    ext = ""
    if op_sys == "windows":
        ext = ".exe"

    output = f"../prg/{op_sys}/{arch}/taskcollect{ext}"
    source = "."
    cmd = [
        "go", "build", "-ldflags=-s -w",
        "-o", output, source
    ]

    env = os.environ
    env["CGO_ENABLED"] = "0"
    env["GOARCH"] = arch
    env["GOOS"] = op_sys

    try:
        print(f"Compiling taskcollect for {op_sys} on {arch}... ", end="", flush=True)
        subprocess.run(
            cmd,
            stdin=sys.stdin,
            stdout=sys.stdout,
            stderr=sys.stderr,
            check=True,
            env=env,
        )
        print("Done")
    except subprocess.CalledProcessError as e:
        print(e)
        sys.exit(1)

def run(argv: list[str]):
    argv = argv[1:]
    if argv[0] == "all":
        for op_sys, archs in platforms.items():
            for arch in archs:
                build(op_sys, arch)
    else:
        for i in argv:
            # TODO: Add error handling?
            i = i.split("/")
            build(i[0], i[1])

def main(argc: int, argv: list[str]):
    if argc == 1:
        print("No arguments provided. See 'help' for more information")
    elif argv[1] == "help":
        print(
            "--- taskcollect: Build help ---\n\n"
            "Supported OS and architectures:\n"
            "- darwin (amd64, arm64)\n"
            "- linux (386, amd64, arm, arm64)\n"
            "- windows (386, arm64)\n"
            "\n"
            "USAGE:\n"
            "    [<<OS>/<ARCH>> ...]\n" # - Provide a valid combination of OS and architecture. Several can be built at once
            "\n"
            "COMMANDS:\n"
            "    all - Build for all platforms (this may take a while)\n"
            "    help - Shows this command\n"
        )
    elif argv[1] == "all":
        run(argv)
    elif argc >= 1:
        run(argv)
    else:
        print("Invalid argument")


if __name__ == "__main__":
    argv = sys.argv
    argc = len(argv)
    main(argc, argv)
    #print(platform.uname())
