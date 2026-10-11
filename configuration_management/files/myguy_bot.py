#!/usr/bin/env python3
"""MyGuy's Telegram admin bot, on the monitoring server (myguy-bot.service).

Uses the alerts' own bot token (long polling: it asks Telegram, nothing
connects to it) and obeys ONE person, the Telegram user
TELEGRAM_ADMIN_USER_ID, in their private chat with the bot only (the alerts
go to a channel, whose posts carry no sender to check).  Everyone else's
commands and button taps are ignored.

  /ban IP          ban for 7 days (fail2ban's manual jail)
  /unban IP        lift every ban on the IP
  /ipstatus IP     the jails that have it banned
  /suspend EMAIL [12h|1d|2d|1w] [REASON]   read-only for a while (none:
                   the next on the ladder: 12h, 1d, 2d, 1w)
  /unsuspend EMAIL
  /pause EMAIL [HOURS]   no sign-in codes to an address being email-bombed
                   (default 24 h, at most 72)
  /unpause EMAIL
  /status EMAIL    blocked? suspended?
  /blocked         the blocks in force
  /investigate IP|EMAIL [HOURS]
  /caught [HOURS]  the accounts the defences point at
  /health          is everything OK right now
  /now             what's happening in the app right now
  /errors [HOURS]  which API calls failed, by endpoint (default 1 h)
  /slow [HOURS]    the slowest API endpoints (default 1 h)
  /bans            every IP banned now, by jail
  /help

The command menu (the / button) is set by the bot itself at start, for
your chat only, so it stays in step with these and nobody else sees it.

Blocking an email is done by hand over SSH on the app server
(`sudo myguy-admin block EMAIL`), never from here.

Changes (ban, unban, suspend, unsuspend) ask first: Confirm or Cancel, valid for
2 minutes.  Your command message is deleted once read (the bot must be a
chat; bots may delete incoming messages there), so a full address doesn't
stay on your phone; replies mask emails (j***@example.com).

The actions run on the app server through `myguy-admin` over SSH, with a
key that can run nothing else, and not block or unblock (myguy_admin.py,
`--bot`).
"""

import html
import json
import os
import re
import secrets
import subprocess
import sys
import time
import urllib.parse
import urllib.request

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from myguy_admin import Refused, parse  # noqa: E402  same checks as the app server

CONFIRM_FOR = 120          # seconds a Confirm button stays valid
POLL_TIMEOUT = 50          # long poll; Telegram answers sooner on news
CHANGES = {"ban", "unban", "suspend", "unsuspend", "pause", "unpause"}
INVESTIGATE_MAX = 3500     # characters of a report in one message
# The command menu, set for the admin's chat only (setMyCommands)
MENU = [
    ("health", "Is everything OK right now"),
    ("now", "What's happening in the app right now"),
    ("errors", "Failed API calls: /errors [hours]"),
    ("slow", "Slowest endpoints: /slow [hours]"),
    ("bans", "IPs banned now, by jail"),
    ("caught", "Accounts to review: /caught [hours]"),
    ("investigate", "Report: /investigate IP|EMAIL [hours]"),
    ("status", "Blocked or suspended? /status EMAIL"),
    ("suspend", "Read-only: /suspend EMAIL [12h|1d|2d|1w] [reason]"),
    ("unsuspend", "Lift a suspension: /unsuspend EMAIL"),
    ("pause", "No codes to an address: /pause EMAIL [hours]"),
    ("unpause", "Codes again: /unpause EMAIL"),
    ("ipstatus", "Where an IP is banned: /ipstatus IP"),
    ("ban", "Ban an IP for 7 days: /ban IP"),
    ("unban", "Lift every ban on an IP: /unban IP"),
    ("blocked", "Blocks in force"),
    ("help", "The admin commands"),
]
HELP = """<b>Admin commands</b>
/ban IP: ban for 7 days
/unban IP: lift every ban on it
/ipstatus IP: where it's banned
/suspend EMAIL [12h|1d|2d|1w] [REASON]: read-only (no length: the next on the ladder, 12h → 1d → 2d → 1w)
/unsuspend EMAIL
/pause EMAIL [HOURS]: no sign-in codes to it (email bombing; default 24 h)
/unpause EMAIL
/status EMAIL: blocked? suspended?
/blocked: blocks in force
/investigate IP|EMAIL [HOURS]
/caught [HOURS]: accounts the defences point at (banned IP, over a limit, many sign-ups)
/health: site, containers, servers, last deploy, errors, right now
/now: posts up, deals and sign-ins today, visitors
/errors [HOURS]: failed API calls by endpoint
/slow [HOURS]: slowest API endpoints
/bans: IPs banned now, by jail
Changes ask you to confirm. Blocking an email is done over SSH on the app server."""


def esc(value):
    return html.escape(str(value), quote=False)


def mask(email):
    local, _, domain = str(email).partition("@")
    return f"{local[:1]}***@{domain}" if domain else "***"


def settings(env=os.environ):
    creds = env.get("CREDENTIALS_DIRECTORY", "")
    return {
        "token": env.get("TELEGRAM_BOT_TOKEN", ""),
        "admin": int(env.get("TELEGRAM_ADMIN_USER_ID") or 0),
        "app": env.get("APP_VPC_IP", ""),
        "key": os.path.join(creds, "admin-key") if creds else env.get("ADMIN_SSH_KEY", ""),
        "known_hosts": env.get("ADMIN_KNOWN_HOSTS", "/etc/myguy-bot/known_hosts"),
        "state": env.get("STATE_DIRECTORY", "/var/lib/myguy-bot"),
    }


# --- Talking to Telegram and the app server -------------------------------------

class Telegram:
    def __init__(self, token):
        self.base = f"https://api.telegram.org/bot{token}/"

    def call(self, method, http_timeout=30, **params):
        data = urllib.parse.urlencode(
            {k: json.dumps(v) if isinstance(v, (dict, list)) else v for k, v in params.items()}).encode()
        with urllib.request.urlopen(self.base + method, data=data, timeout=http_timeout) as r:
            body = json.loads(r.read())
        if not body.get("ok"):
            raise RuntimeError(f"telegram {method}: {body.get('description')}")
        return body.get("result")


def admin_over_ssh(s):
    """Runs one myguy-admin command on the app server, returning its JSON."""
    def run(command):
        result = subprocess.run(
            ["ssh", "-i", s["key"], "-o", "IdentitiesOnly=yes", "-o", "BatchMode=yes",
             "-o", "StrictHostKeyChecking=yes", "-o", f"UserKnownHostsFile={s['known_hosts']}",
             "-o", "ConnectTimeout=10", f"myguy-bot@{s['app']}", command],
            capture_output=True, text=True, timeout=60)
        try:
            return json.loads(result.stdout.strip().splitlines()[-1])
        except (ValueError, IndexError):
            return {"ok": False, "error": (result.stderr.strip() or "no answer from the app server")[:200]}
    return run


def security(args):
    """A myguy-security report (investigate, caught, health, now) as plain
    text; args are checked by the bot first."""
    result = subprocess.run(["/usr/local/bin/myguy-security", *args], capture_output=True, text=True, timeout=120)
    return (result.stdout or result.stderr).strip() or "(nothing)"


# --- The bot ----------------------------------------------------------------------

class Bot:
    def __init__(self, s, telegram, admin, reports=security, now=time.time):
        self.s, self.tg, self.admin, self.reports, self.now = s, telegram, admin, reports, now
        self.pending = {}  # token -> (action, args, command, asked at); full emails stay in memory

    def say(self, text, **extra):
        return self.tg.call("sendMessage", chat_id=self.s["admin"], text=text, parse_mode="HTML",
                            disable_web_page_preview="true", **extra)

    def allowed(self, chat_id, user_id):
        # A private chat's id is the other person's user id
        return bool(self.s["admin"]) and chat_id == self.s["admin"] and user_id == self.s["admin"]

    def handle(self, update):
        if "callback_query" in update:
            return self.on_button(update["callback_query"])
        message = update.get("message") or {}
        text = message.get("text") or ""
        if not text.startswith("/"):
            return
        chat_id, user_id = message.get("chat", {}).get("id"), message.get("from", {}).get("id")
        if not self.allowed(chat_id, user_id):
            print(f"bot event=ignored user={user_id} chat={chat_id}", flush=True)
            return
        try:
            # Read: take the full address out of the chat
            self.tg.call("deleteMessage", chat_id=chat_id, message_id=message["message_id"])
        except Exception as e:  # noqa: BLE001  not an admin, or too old
            print(f"bot: couldn't delete the command: {e}", flush=True)
        self.on_command(text)

    def on_command(self, text):
        words = text.split()
        name = words[0][1:].split("@")[0].lower()
        rest = words[1:]
        if name in ("help", "start"):
            return self.say(HELP)
        if name == "investigate":
            return self.on_investigate(rest)
        if name == "caught":
            return self.on_caught(rest)
        if name in ("health", "now"):
            return self.report([name]) if not rest else self.say(f"⚠️ usage: /{name}")
        if name in ("errors", "slow"):
            if len(rest) > 1 or (rest and not rest[0].isdigit()):
                return self.say(f"⚠️ usage: /{name} [HOURS]")
            return self.report([name] + (["--hours", rest[0]] if rest else []))
        try:
            action, args = parse(" ".join([name] + rest), bot=True)
        except Refused as e:
            return self.say(f"⚠️ {esc(e)}")
        command = command_for(action, args)
        if action not in CHANGES:
            return self.say(self.describe(action, args, self.admin(command)))
        token = secrets.token_hex(8)
        self.pending[token] = (action, args, command, self.now())
        self.say(f"{self.question(action, args)}\n<i>Confirm within 2 minutes.</i>", reply_markup={
            "inline_keyboard": [[{"text": "Confirm", "callback_data": f"ok:{token}"},
                                 {"text": "Cancel", "callback_data": f"no:{token}"}]]})

    def on_button(self, query):
        chat_id = (query.get("message") or {}).get("chat", {}).get("id")
        user_id = query.get("from", {}).get("id")
        if not self.allowed(chat_id, user_id):
            print(f"bot event=ignored_button user={user_id} chat={chat_id}", flush=True)
            self.tg.call("answerCallbackQuery", callback_query_id=query["id"], text="Not for you.")
            return
        answer, _, token = (query.get("data") or "").partition(":")
        pending = self.pending.pop(token, None)
        message_id = query["message"]["message_id"]
        self.tg.call("answerCallbackQuery", callback_query_id=query["id"])
        if not pending or self.now() - pending[3] > CONFIRM_FOR:
            text = "⌛ Expired: send the command again."
        elif answer != "ok":
            text = f"Cancelled: {self.question(pending[0], pending[1])}"
        else:
            action, args, command, _ = pending
            result = self.admin(command)
            print(f"bot event={action} target={args.get('ip') or mask(args.get('email'))} "
                  f"ok={str(result.get('ok')).lower()}", flush=True)
            text = self.describe(action, args, result)
        self.tg.call("editMessageText", chat_id=chat_id, message_id=message_id, text=text, parse_mode="HTML")

    def on_caught(self, rest):
        if len(rest) > 1 or (rest and not rest[0].isdigit()):
            return self.say("⚠️ usage: /caught [HOURS]")
        self.report(["caught"] + (["--hours", rest[0]] if rest else []))

    def report(self, args):
        report = self.reports(args)
        if len(report) > INVESTIGATE_MAX:
            report = report[:INVESTIGATE_MAX] + "\n… (cut: run it on the monitoring server for all of it)"
        self.say(f"<pre>{esc(report)}</pre>")

    def on_investigate(self, rest):
        if not rest or len(rest) > 2 or (len(rest) == 2 and not rest[1].isdigit()):
            return self.say("⚠️ usage: /investigate IP|EMAIL [HOURS]")
        target = rest[0]
        try:
            target = parse(f"ipstatus {target}")[1]["ip"]
        except Refused:
            try:
                target = parse(f"status {target}")[1]["email"]
            except Refused:
                return self.say("⚠️ not an IP or an email address")
        self.report(["investigate", target] + (["--hours", rest[1]] if len(rest) == 2 else []))

    @staticmethod
    def question(action, args):
        if action == "ban":
            return f"Ban <code>{esc(args['ip'])}</code> for 7 days?"
        if action == "unban":
            return f"Lift every ban on <code>{esc(args['ip'])}</code>?"
        who = f"<code>{esc(mask(args['email']))}</code>"
        if action == "suspend":
            how = f"for {LENGTH_NAMES[args['hours']]}" if args["hours"] else "for the next length on the ladder"
            why = f" (reason: {esc(args['reason'])})" if args.get("reason") else ""
            return f"Suspend {who} {how}{why}?"
        if action == "pause":
            return f"Stop sign-in codes to {who} for {args['hours']} hours?"
        if action == "unpause":
            return f"Send sign-in codes to {who} again?"
        return f"Lift the suspension of {who}?"

    @staticmethod
    def describe(action, args, result):
        if not result.get("ok"):
            return f"❌ {esc(result.get('error') or 'failed')}"
        if action == "ban":
            return f"🚫 Banned <code>{esc(args['ip'])}</code> for 7 days."
        if action == "unban":
            return f"✅ Lifted every ban on <code>{esc(args['ip'])}</code>."
        if action == "ipstatus":
            jails = result.get("jails") or []
            return (f"<code>{esc(args['ip'])}</code> is banned in: {esc(', '.join(jails))}" if jails
                    else f"<code>{esc(args['ip'])}</code> isn't banned.")
        if action == "bans":
            jails = {j: ips for j, ips in (result.get("jails") or {}).items() if ips}
            if not jails:
                return "No IP is banned right now."
            lines = [f"<b>Banned now</b>: {sum(len(i) for i in jails.values())} IPs"]
            for jail, ips in sorted(jails.items(), key=lambda x: -len(x[1])):
                shown = ", ".join(f"<code>{esc(ip)}</code>" for ip in ips[:8])
                lines.append(f"• {esc(jail)} ({len(ips)}): {shown}" + (" …" if len(ips) > 8 else ""))
            return "\n".join(lines)
        if action == "blocked":
            blocks = result.get("blocks") or []
            if not blocks:
                return "No email is blocked."
            return "<b>Blocked</b>\n" + "\n".join(
                f"• <code>{esc(mask(b['email']))}</code> {until(b)}" for b in blocks[:30])
        who = f"<code>{esc(mask(args['email']))}</code>"
        if action == "suspend":
            step = result.get("step") or 1
            text = (f"⏸ Suspended {who} until {esc((result.get('until') or '')[:16].replace('T', ' '))} UTC "
                    f"({LENGTH_NAMES.get(result.get('hours'), str(result.get('hours')) + ' hours')}, {ordinal(step)} in a row). "
                    "Read-only; their partners are told within a minute.")
            if step >= 4:
                text += "\nThat's 4 or more in a row: consider a block (over SSH on the app server)."
            return text
        if action == "unsuspend":
            return f"✅ Lifted the suspension of {who}. It still counts towards the next one."
        if action == "pause":
            return (f"⏸ No sign-in codes to {who} until {esc((result.get('until') or '')[:16].replace('T', ' '))} UTC. "
                    "Whoever asks is told a code was sent.")
        if action == "unpause":
            return f"✅ Sign-in codes go to {who} again."
        lines = [f"{who} is blocked {until(result)}." if result.get("blocked") else f"{who} isn't blocked."]
        s = result.get("suspension")
        if s and s.get("suspended"):
            lines.append(f"Suspended until {esc((s.get('until') or '')[:16].replace('T', ' '))} UTC"
                         + (f" ({esc(s['reason'])})" if s.get("reason") else "") + f", {ordinal(s.get('in_row') or 1)} in a row.")
        elif s:
            nxt = LENGTH_NAMES.get(s.get("next_hours"), "12 hours")
            lines.append(f"Not suspended ({s.get('in_row') or 0} in a row; the next would be {nxt}).")
        if result.get("codes_paused_until"):
            lines.append(f"Sign-in codes paused until {esc(result['codes_paused_until'][:16].replace('T', ' '))} UTC.")
        return "\n".join(lines)


def command_for(action, args):
    """The myguy-admin command, from the checked values (never raw input)."""
    if action == "suspend":
        length = HOUR_TOKENS[args["hours"]] if args["hours"] else ""
        return " ".join(w for w in (action, args["email"], length, args["reason"]) if w)
    if action == "pause":
        return f"pause {args['email']} {args['hours']}"
    target = args.get("ip") or args.get("email")
    return f"{action} {target}" if target else action


LENGTH_NAMES = {12: "12 hours", 24: "1 day", 48: "2 days", 168: "1 week"}
HOUR_TOKENS = {12: "12h", 24: "1d", 48: "2d", 168: "1w"}


def ordinal(n):
    return f"{n}{'th' if 10 <= n % 100 <= 20 else {1: 'st', 2: 'nd', 3: 'rd'}.get(n % 10, 'th')}"


def until(block):
    when = block.get("until")
    reason = f" ({esc(block['reason'])})" if block.get("reason") else ""
    return (f"until {esc(when[:10])}" if when else "for good") + reason


# --- Running -----------------------------------------------------------------------

def load_offset(path):
    try:
        with open(path) as f:
            return int(f.read().strip() or 0)
    except (OSError, ValueError):
        return 0


def save_offset(path, offset):
    with open(path + ".tmp", "w") as f:
        f.write(str(offset))
    os.replace(path + ".tmp", path)


def set_menu(tg, admin):
    """The / menu, in the admin's chat only; a failure just leaves it as it was."""
    try:
        tg.call("setMyCommands", commands=[{"command": c, "description": d} for c, d in MENU],
                scope={"type": "chat", "chat_id": admin})
    except Exception as e:  # noqa: BLE001
        print(f"myguy-bot: couldn't set the command menu: {e}", flush=True)


def main():
    s = settings()
    missing = [k for k in ("token", "admin", "app", "key") if not s[k]]
    if missing:
        print(f"myguy-bot: not configured ({', '.join(missing)} missing); idle", flush=True)
        while True:  # stay up quietly so systemd doesn't restart-loop
            time.sleep(3600)
    tg = Telegram(s["token"])
    set_menu(tg, s["admin"])
    bot = Bot(s, tg, admin_over_ssh(s))
    offset_file = os.path.join(s["state"], "offset")
    offset = load_offset(offset_file)
    print("myguy-bot: listening", flush=True)
    while True:
        try:
            updates = tg.call("getUpdates", http_timeout=POLL_TIMEOUT + 10, timeout=POLL_TIMEOUT,
                              offset=offset, allowed_updates=["message", "callback_query"])
        except Exception as e:  # noqa: BLE001  network, Telegram hiccup
            print(f"myguy-bot: getUpdates failed: {e}", flush=True)
            time.sleep(10)
            continue
        for update in updates:
            offset = update["update_id"] + 1
            save_offset(offset_file, offset)  # before acting: never act twice
            try:
                bot.handle(update)
            except Exception as e:  # noqa: BLE001
                print(f"myguy-bot: update failed: {e}", flush=True)


if __name__ == "__main__":
    main()
