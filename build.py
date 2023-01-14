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

def build(os_name: str, arch: str):
    version_line = 'const tcVersion = "TaskCollect 0.1.0 (build {0})"\n'
    try:
        version = subprocess.check_output("git describe --tags --abbrev=0", shell=True, stderr=subprocess.DEVNULL)
        version = str(version)[3:-3]
    except subprocess.CalledProcessError:
        version = "0.1.0"
    commit = subprocess.check_output("git rev-parse HEAD", shell=True, stderr=subprocess.DEVNULL)
    version_line = version_line.replace("0.1.0", str(version)).replace("{0}", str(commit)[2:-3])
    with open("src/version.go", "w") as f:
        f.write("package main\n\n" + version_line)

    ext = ""
    if os_name == "windows":
        ext = ".exe"

    cwd = Path.cwd().joinpath("src")

    output = f"../prg/{os_name}/{arch}/taskcollect{ext}"
    source = "."
    cmd = ["go", "build", "-ldflags=-s -w", "-o", output, source]

    env = os.environ
    env["CGO_ENABLED"] = "0"
    env["GOARCH"] = arch
    env["GOOS"] = os_name

    try:
        print(f"Compiling taskcollect for {os_name} on {arch}... ", end="", flush=True)
        subprocess.run(
            cmd,
            stdin=sys.stdin,
            stdout=sys.stdout,
            stderr=sys.stderr,
            check=True,
            env=env,
            cwd=cwd
        )
        print("Done")
    except subprocess.CalledProcessError as e:
        print(e)
        sys.exit(1)

def run(argv: list[str]):
    argv = argv[1:]
    if argv[0] == "all":
        for os_name, archs in platforms.items():
            for arch in archs:
                build(os_name, arch)
    else:
        for i in argv:
            # TODO: Add error handling?
            i = i.split("/")
            build(i[0], i[1])


def update_res_files():
    from distutils.dir_util import copy_tree
    import shutil

    res_src = Path.cwd().joinpath("res")
    res_src = Path.resolve(res_src)
    tmpl_src = res_src.joinpath("templates")

    res_dst = Path.home().joinpath("res/taskcollect")
    tmpl_dst = res_dst.joinpath("templates")

    print("Begin to copy assets...")

    # Copy over template files
    src = tmpl_src
    rel_path = Path.relative_to(src, tmpl_src)
    dst = tmpl_dst.joinpath(rel_path)
    copy_tree(str(src), str(dst))
    print(f"  Copied {src} -> {dst}")

    # NOTE: shell=True must be set for the command to work on Windows systems.
    # This is due to the Sass program being invoked via `sass.bat` which is not recognised
    # by subprocess unless shell=True. 

    subprocess.run(
        "sass ./src/styles/styles.scss ./res/styles.css --no-source-map",
        shell=True
    )
    print("Compiled SCSS files into CSS file")

    res_dst.joinpath("icons").mkdir(exist_ok=True)

    assets = [
        "styles.css", "script.js",
        "taskcollect-logo.svg", "taskcollect-wordmark.svg",
        "icons/apple-touch-icon.png", "icons/favicon.ico",
        "icons/icon-192.png", "icons/icon-512.png", "icons/icon.svg"
    ]

    for asset in assets:
        src = res_src.joinpath(asset)
        dst = res_dst.joinpath(asset)
        shutil.copy(src, dst)
        print(f"  Copied {src} -> {dst}")


def print_help():
    cmd = "python3 build.py"
    if platform.system().lower() == "windows":
        cmd = "py build.py"
    print(
        "--- TaskCollect: Build help ---\n\n"
        "Supported OS and architectures:\n"
        "  - darwin  (amd64, arm64)\n"
        "  - linux   (386, amd64, arm, arm64)\n"
        "  - windows (386, amd64)\n\n"
        "Invoke the script without any arguments to automatically build for your system.\n\n"
        "USAGE:\n"
        f"    {cmd} [<<OS>/<ARCH>> ...]\n" # - Provide a valid combination of OS and architecture. Several can be built at once
        "\n"
        "OPTIONS:\n"
        "    -u     Build while also copying across resource files\n"
        "    -U     Only copy resource files, do not build\n"
        "\n"
        "COMMANDS:\n"
        "    all    Build for all platforms (this may take a while)\n"
        "    help   Shows this command\n"
    )

def main(argc: int, argv: list[str]):
    if "-u" in argv:
        update_res_files()
        print("Files have been copied.")
        argv.remove("-u")
        argc -= 1
    elif "-U" in argv:
        update_res_files()
        print("Files have been copied. Not building TaskCollect due to no-build option")
        sys.exit(0)

    if argc == 1:
        os_name = platform.system().lower()
        arch = platform.machine().lower()
        if arch in ["i386", "x86"]:
            arch = "386"
        elif arch in ["x64", "x86_64"]:
            arch = "amd64"
        print(f"Automatically building TaskCollect for the host system: {os_name}/{arch}")
        print("See 'help' for more information")
        argv.append(f"{os_name}/{arch}")
        run(argv)
    elif argv[1] == "help":
        print_help()
    elif argv[1] == "all" or argc >= 1:
        run(argv)
    else:
        print("Invalid argument")


if __name__ == "__main__":
    argv = sys.argv
    argc = len(argv)
    main(argc, argv)
