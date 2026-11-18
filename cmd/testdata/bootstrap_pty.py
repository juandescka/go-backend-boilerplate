"""Exercise the operator CLI on a pseudo-terminal without printing credentials."""
import errno
import json
import os
import pty
import select
import signal
import subprocess
import sys
import termios
import time

account = json.load(sys.stdin)
password = account["password"].encode()
master, slave = pty.openpty()
process = subprocess.Popen(sys.argv[1:], stdin=slave, stdout=slave, stderr=slave)
transcript = bytearray()
deadline = time.monotonic() + 15


def receive():
    remaining = deadline - time.monotonic()
    if remaining <= 0 or not select.select([master], [], [], remaining)[0]:
        raise TimeoutError()
    try:
        return os.read(master, 4096)
    except OSError as error:
        if error.errno == errno.EIO:
            return b""
        raise


def wait_for_prompt(prompt, hidden=False):
    start = len(transcript)
    while prompt not in transcript[start:]:
        chunk = receive()
        if not chunk:
            raise EOFError()
        transcript.extend(chunk)
    if hidden and termios.tcgetattr(slave)[3] & termios.ECHO:
        raise ValueError("terminal echo is enabled")

def answer(prompt, value, hidden=False):
    wait_for_prompt(prompt, hidden)
    os.write(master, value + b"\n")


try:
    if account.get("mode") != "reject":
        answer(b"Administrator email: ", account["email"].encode())
        answer(b"Administrator display name: ", b"Disposable administrator")
        if account.get("mode") == "interrupt":
            wait_for_prompt(b"New password (input hidden): ", hidden=True)
            process.send_signal(signal.SIGINT)
            if process.wait(timeout=2) != 130 or not termios.tcgetattr(slave)[3] & termios.ECHO:
                raise ValueError("interrupted bootstrap did not restore terminal echo")
            print(json.dumps({"code": 130, "result": "interrupted"}))
            sys.exit(0)
        answer(b"New password (input hidden): ", password, hidden=True)
        if account.get("mode") != "invalid-password":
            answer(b"Confirm password (input hidden): ", password, hidden=True)
    os.close(slave)
    slave = None
    while chunk := receive():
        transcript.extend(chunk)
    status = process.wait(timeout=2)
    if password in transcript or b"$argon2id$" in transcript:
        raise ValueError("credential appeared in terminal output")
    messages = {
        b"Initial platform administrator created.": "created",
        b"password must contain at least 15 characters": "password-policy",
        b"bootstrap has already been recorded": "already",
        b"email already belongs to an existing account": "existing",
        b"could not create the initial platform assignment": "rollback",
        b"migration version of at least 15": "migration",
        b"requires the development environment": "environment",
        b"--target must match": "target",
    }
    print(json.dumps({"code": status, "result": next(
        (label for message, label in messages.items() if message in transcript), "unexpected"
    )}))
except (OSError, ValueError, TimeoutError, EOFError, subprocess.TimeoutExpired):
    sys.exit("bootstrap terminal verification failed (transcript withheld)")
finally:
    if process.poll() is None:
        process.kill()
        process.wait()
    if slave is not None:
        os.close(slave)
    os.close(master)
