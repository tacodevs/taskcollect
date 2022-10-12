#!/usr/bin/python3

from os import *
import sys
import subprocess

platforms = """
	386/linux
	386/windows
	amd64/darwin
	amd64/linux
	amd64/windows
	arm/linux
	arm64/darwin
	arm64/linux
"""

for platform in platforms.split():
	platform = platform.split("/")
	arch = platform[0]
	os = platform[1]

	ext = ""
	if os == "windows":
		ext = ".exe"

	cmd = [
		"go", "build", "-ldflags=-s -w",
		"-o", f"../prg/{arch}/{os}/taskcollect{ext}", ".",
	]

	env = environ
	env["CGO_ENABLED"] = "0"
	env["GOARCH"] = arch
	env["GOOS"] = os

	try:
		subprocess.run(
			cmd,
			stdin=sys.stdin,
			stdout=sys.stdout,
			stderr=sys.stderr,
			check=True,
			env=env,
		)
	except subprocess.CalledProcessError:
		sys.exit(1)
