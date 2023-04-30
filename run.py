#!/usr/bin/python3

import os
from pathlib import Path
import platform
import subprocess
import sys

platforms = {
    "darwin": ["amd64", "arm64"],
    "linux": ["386", "amd64", "arm", "arm64"],
    "windows": ["386", "amd64"]
}

def run(os_name: str, arch: str, argv: list[str]):
    ext = ""
    if os_name == "windows":
        ext = ".exe"

    cwd = Path.cwd()

    executable = f"./prg/{os_name}/{arch}/taskcollect{ext}"
    argv[0] = executable
    try:
        subprocess.run(
            argv,
            stdin=sys.stdin,
            stdout=sys.stdout,
            stderr=sys.stderr,
            check=True,
            cwd=cwd
        )
        print("Done")
    except subprocess.CalledProcessError as e:
        print(e)
        sys.exit(1)

def main(argc: int, argv: list[str]):
    os_name = platform.system().lower()
    arch = platform.machine().lower()
    if arch in ["i386", "x86"]:
        arch = "386"
    elif arch in ["x64", "x86_64"]:
        arch = "amd64"
    run(os_name, arch, argv)

if __name__ == "__main__":
    argv = sys.argv
    argc = len(argv)
    main(argc, argv)
