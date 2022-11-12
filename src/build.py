#!/usr/bin/python3

import os
import platform
import subprocess
import sys


platforms = {
    "darwin": ["amd64", "arm64"],
    "linux": ["386", "amd64", "arm", "arm64"],
    "windows": ["386", "amd64"]
}

def build(os_name: str, arch: str):
    ext = ""
    if os_name == "windows":
        ext = ".exe"

    output = f"../prg/{os_name}/{arch}/taskcollect{ext}"
    source = "."
    cmd = [
        "go", "build", "-ldflags=-s -w",
        "-o", output, source
    ]

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
    import glob
    from pathlib import Path
    import shutil

    res_src = Path.cwd().joinpath("..", "res")
    res_src = Path.resolve(res_src)
    tmpl_src = res_src.joinpath("./templates/")

    res_dst = Path.home().joinpath("./res/taskcollect/")
    tmpl_dst = res_dst.joinpath("./templates/")

    # Copy over template files
    src = tmpl_src
    rel_path = Path.relative_to(src, tmpl_src)
    dst = tmpl_dst.joinpath(rel_path)
    copy_tree(str(src), str(dst))
    print(f"Copied {src} -> {dst}")

    # Copy CSS stylesheet
    src = res_src.joinpath("styles.css")
    dst = res_dst.joinpath("styles.css")
    shutil.copy(src, dst)
    print(f"Copied {src} -> {dst}")



def main(argc: int, argv: list[str]):
    if "-u" in argv:
        update_res_files()
        print("Files have been copied.")
        argv.remove("-u")
        argc -= 1
    elif "-U" in argv:
        update_res_files()
        print("Files have been copied. Not building due to no-build option")
        sys.exit(0)

    if argc == 1:
        os_name = platform.system().lower()
        arch = platform.machine().lower()
        if arch in ["i386", "x86"]:
            arch = "386"
        elif arch in ["x64", "x86_64"]:
            arch = "amd64"
        print(f"Automatically building taskcollect for the host system: {os_name}/{arch}")
        print("See 'help' for more information")
        argv.append(f"{os_name}/{arch}")
        run(argv)
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
            "OPTIONS:\n"
            "    -u - Build while also copying across resource files\n"
            "    -U - Only copy resource files, do not build\n"
            "\n"
            "COMMANDS:\n"
            "    all - Build for all platforms (this may take a while)\n"
            "    help - Shows this command\n"
        )
    elif argv[1] == "all" or argc >= 1:
        run(argv)
    else:
        print("Invalid argument")


if __name__ == "__main__":
    argv = sys.argv
    argc = len(argv)
    main(argc, argv)
