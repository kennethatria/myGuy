#!/usr/bin/env python3
"""Admin actions on the app server: ban or unban an IP, block or unblock an
email.  The only thing the Telegram bot (myguy-bot on the monitoring server)
can run here: its SSH key is restricted to

    command="sudo -n /usr/local/bin/myguy-admin --bot \"$SSH_ORIGINAL_COMMAND\""

so whatever the bot sends arrives as ONE argument, which is split and
checked here word by word.  Nothing reaches a shell.  `--bot` (fixed by that
forced command and by the bot's sudoers rule) leaves out block and unblock:
blocking an email is done by hand, over SSH here:

    sudo myguy-admin block EMAIL [DAYS] [REASON]

  ban IP                       7 days in fail2ban's manual jail
  unban IP                     out of every jail (automatic bans too)
  ipstatus IP                  the jails that have it banned
  bans                         every IP banned now, by jail
  block EMAIL [DAYS] [REASON]  DAYS 0 or none: for good
  unblock EMAIL
  suspend EMAIL [LENGTH] [REASON]  read-only for 12h, 1d, 2d or 1w; none:
                               the next on the ladder (12h, 1d, 2d, 1w)
  unsuspend EMAIL
  status EMAIL                 blocked? suspended?
  blocked                      the blocks in force

Prints one JSON object ({"ok": true, ...} or {"ok": false, "error": ...})
and logs each change as `admin event=... by=telegram|ssh:USER` (journal →
Loki, job="admin").  Runs as root through sudo: fail2ban's socket and the .env
holding INTERNAL_API_KEY are root's.
"""

import ast
import ipaddress
import json
import os
import re
import subprocess
import sys
import syslog
import urllib.error
import urllib.parse
import urllib.request

ENV_FILE = "/opt/myguy/.env"
BACKEND = "http://127.0.0.1:8080/internal/v1"
FAIL2BAN = "/usr/bin/fail2ban-client"
MAX_DAYS = 3650
REASON_MAX = 200
# Ours: the NodeBalancer (every visitor seems to come from it without the
# PROXY header) and the VPC.  Banning them would take the site down.
PROTECTED_NETS = [ipaddress.ip_network(n) for n in ("192.168.255.0/24", "10.0.0.0/24")]
EMAIL = re.compile(r"^[A-Za-z0-9._%+-]{1,64}@[A-Za-z0-9.-]{1,190}\.[A-Za-z]{2,24}$")
REASON = re.compile(r"^[\w .,:;!?'()/#&+-]*$")
# Suspension lengths, in hours (the backend accepts exactly these)
LENGTHS = {"12h": 12, "1d": 24, "24h": 24, "2d": 48, "48h": 48, "1w": 168, "7d": 168}


class Refused(Exception):
    """A request that isn't allowed; the message is for the person."""


def ip_arg(text):
    try:
        ip = ipaddress.ip_address(text)
    except ValueError:
        raise Refused(f"not an IP address: {text[:60]}")
    if not ip.is_global or any(ip in net for net in PROTECTED_NETS):
        raise Refused(f"{ip} is private or ours; it can't be banned")
    return str(ip)


def email_arg(text):
    email = text.strip().lower()
    if len(email) > 254 or not EMAIL.match(email):
        raise Refused(f"not an email address: {text[:60]}")
    return email


def days_arg(text):
    if not text.isdigit() or int(text) > MAX_DAYS:
        raise Refused(f"days must be a whole number from 0 to {MAX_DAYS}")
    return int(text)


def reason_arg(words):
    reason = " ".join(words)
    if len(reason) > REASON_MAX or not REASON.match(reason):
        raise Refused(f"the reason may hold letters, digits and simple punctuation, up to {REASON_MAX} characters")
    return reason


def parse(command, bot=False):
    """(action, args) from the one-line command, or Refused."""
    words = command.split()
    if not words:
        raise Refused("no command")
    action, rest = words[0].lower(), words[1:]
    if bot and action in ("block", "unblock"):  # a deliberate step, over SSH
        raise Refused(f"{action} is done over SSH on the app server: sudo myguy-admin {action} EMAIL")
    if action in ("ban", "unban", "ipstatus"):
        if len(rest) != 1:
            raise Refused(f"usage: {action} IP")
        return action, {"ip": ip_arg(rest[0])}
    if action == "block":
        if not rest:
            raise Refused("usage: block EMAIL [DAYS] [REASON]")
        args = {"email": email_arg(rest[0]), "days": 0, "reason": ""}
        more = rest[1:]
        if more and more[0].isdigit():
            args["days"] = days_arg(more[0])
            more = more[1:]
        args["reason"] = reason_arg(more)
        return action, args
    if action == "suspend":
        if not rest:
            raise Refused("usage: suspend EMAIL [12h|1d|2d|1w] [REASON]")
        args = {"email": email_arg(rest[0]), "hours": 0, "reason": ""}
        more = rest[1:]
        if more and more[0].lower() in LENGTHS:
            args["hours"] = LENGTHS[more[0].lower()]
            more = more[1:]
        elif more and re.fullmatch(r"\d+[hdw]?", more[0].lower()):
            raise Refused("a suspension lasts 12h, 1d, 2d or 1w (none: the next on the ladder)")
        args["reason"] = reason_arg(more)
        return action, args
    if action in ("unblock", "unsuspend", "status"):
        if len(rest) != 1:
            raise Refused(f"usage: {action} EMAIL")
        return action, {"email": email_arg(rest[0])}
    if action in ("blocked", "bans"):
        if rest:
            raise Refused(f"usage: {action}")
        return action, {}
    raise Refused(f"unknown command: {action[:30]}")


# --- Doing it ------------------------------------------------------------------

def internal_key(path=ENV_FILE):
    with open(path) as f:
        for line in f:
            if line.startswith("INTERNAL_API_KEY="):
                return line.split("=", 1)[1].strip()
    raise RuntimeError("INTERNAL_API_KEY isn't in " + path)


def backend(method, path, body=None, key=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(BACKEND + path, data=data, method=method, headers={
        "X-Internal-API-Key": key or internal_key(), "Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            return r.status, json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.loads(e.read() or b"{}")
        except ValueError:
            return e.code, {}


def fail2ban(*args):
    result = subprocess.run([FAIL2BAN, *args], capture_output=True, text=True, timeout=30)
    if result.returncode != 0:
        raise RuntimeError((result.stderr or result.stdout).strip()[:200] or "fail2ban-client failed")
    return result.stdout.strip()


def run(action, args, f2b=fail2ban, call=backend, by="telegram"):
    """The result as a dict for the bot."""
    if action == "ban":
        f2b("set", "manual", "banip", args["ip"])
        return {"ok": True, "ip": args["ip"], "days": 7}
    if action == "unban":
        f2b("unban", args["ip"])
        return {"ok": True, "ip": args["ip"]}
    if action == "bans":
        # fail2ban prints [{'sshd': ['1.2.3.4']}, {'manual': []}, ...]
        jails = {}
        for entry in ast.literal_eval(f2b("banned") or "[]"):
            jails.update({str(j): [str(ip) for ip in ips] for j, ips in entry.items()})
        return {"ok": True, "jails": jails}
    if action == "ipstatus":
        # fail2ban prints the jails holding the IP, e.g. [['tripwire']]
        jails = re.findall(r"'([\w-]+)'", f2b("banned", args["ip"]))
        return {"ok": True, "ip": args["ip"], "jails": jails}
    if action == "block":
        status, body = call("POST", "/email-blocks", {**args, "by": by})
    elif action == "unblock":
        status, body = call("DELETE", "/email-blocks/" + urllib.parse.quote(args["email"]))
    elif action == "suspend":
        status, body = call("POST", "/suspensions", {**args, "by": by})
    elif action == "unsuspend":
        status, body = call("DELETE", "/suspensions/" + urllib.parse.quote(args["email"]))
    elif action == "status":
        status, body = call("GET", "/email-blocks/" + urllib.parse.quote(args["email"]))
        if status < 400:
            # No account (only an address) has no suspension to show
            s_status, suspension = call("GET", "/suspensions/" + urllib.parse.quote(args["email"]))
            body = {**body, "suspension": suspension if s_status < 400 else None}
    else:
        status, body = call("GET", "/email-blocks")
    if status >= 400:
        return {"ok": False, "error": body.get("error") or f"backend answered {status}"}
    return {"ok": True, **body}


CHANGES = {"ban", "unban", "block", "unblock", "suspend", "unsuspend"}


def main(argv=None):
    argv = sys.argv[1:] if argv is None else argv
    bot = argv[:1] == ["--bot"]
    command = " ".join(argv[1:] if bot else argv)
    by = "telegram" if bot else "ssh:" + (os.environ.get("SUDO_USER") or "root")
    syslog.openlog("myguy-admin")
    try:
        action, args = parse(command, bot=bot)
        result = run(action, args, by=by)
    except Refused as e:
        result, action, args = {"ok": False, "error": str(e)}, "refused", {}
    except Exception as e:  # noqa: BLE001
        result, action, args = {"ok": False, "error": f"failed: {e}"[:300]}, "failed", {}
    if action in CHANGES or not result["ok"]:
        target = args.get("ip") or args.get("email") or command[:80]
        extra = (f" days={args['days']}" if action == "block"
                 else f" hours={result.get('hours', args['hours'])}" if action == "suspend" and result["ok"] else "")
        syslog.syslog(f"admin event={action} target={target}{extra} ok={str(result['ok']).lower()} by={by}")
    print(json.dumps(result))
    return 0 if result["ok"] else 1


if __name__ == "__main__":
    sys.exit(main())
