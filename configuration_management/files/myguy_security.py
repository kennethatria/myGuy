#!/usr/bin/env python3
"""Security checks, summaries and IP investigation for MyGuy.

Runs on the monitoring server (installed by monitoring.yml as
/usr/local/bin/myguy-security) against its Loki and Prometheus:

  myguy-security check            every 5 minutes: attacks that need a
                                  decision, unexpected SSH logins and bans
                                  that may have hit a real user go to
                                  Telegram; logs "check ok" (the heartbeat
                                  Grafana watches)
  myguy-security summary          08:00 Amsterdam time: the daily summary,
                                  plus the weekly one on Saturdays
  myguy-security daily|weekly     one of them, now
  myguy-security investigate IP   everything one IP did, in the terminal
  myguy-security investigate EMAIL  every sign-in step for one address

Add --dry-run to print messages instead of sending them.

Telegram messages are HTML.  Every value that came from a request (paths,
user agents, process names) is escaped and put in <code>, so text an
attacker sent is never a link or formatting.  Settings come from the
environment, or from /etc/myguy-security/settings.env when unset.
"""

import argparse
import datetime as dt
import html
import ipaddress
import json
import os
import re
import sys
import urllib.parse
import urllib.request
from collections import Counter, defaultdict
from zoneinfo import ZoneInfo

TZ = ZoneInfo("Europe/Amsterdam")
SETTINGS_FILE = "/etc/myguy-security/settings.env"

MIN = 60 * 10**9  # nanoseconds, as Loki counts time
HOUR = 60 * MIN
DAY = 24 * HOUR

# What makes an attacking IP worth a message (see the README runbook).
WINDOW = 10 * MIN            # each check looks at the last 10 minutes
WAF_HITS_UNBANNED = 5        # flagged requests; fail2ban bans at 3
WAF_GRACE = 5 * MIN          # time fail2ban gets to ban before we ask you
SIGN_IN_CALLS = 10           # sign-in calls; real use peaks at ~6, ban at 15
EPISODE_GAP = 30 * MIN       # quiet this long and an attack has ended
TOP_IPS = 5                  # IPs listed in one message
HISTORY_DAYS = 15            # saved summaries (the weekly compares 2 weeks)
DEPLOY_LONGEST = 90 * MIN    # a deploy whose end marker never came

# OWASP CRS rule files by the first three digits of a rule id
CATEGORIES = {
    "913": "scanner", "920": "protocol", "921": "protocol attack",
    "930": "file access", "931": "remote file inclusion",
    "932": "code execution", "933": "PHP injection", "934": "code injection",
    "941": "XSS", "942": "SQL injection", "943": "session fixation",
    "944": "Java/Log4Shell",
}
# Real attacks rather than odd clients: the "Serious attack" set
SERIOUS = {"931", "932", "933", "934", "941", "942", "944"}
# Score evaluation, not a finding of its own
EVALUATION = {"949", "959", "980"}
FLAGGED_RULE = "949110"      # the request's score crossed the threshold

FALCO_CRITICAL = {"Emergency", "Alert", "Critical", "Error"}
SITE_DOWN_HELP = "Site down alerts say when a deploy ran in the last 30 minutes."


# --- Settings -------------------------------------------------------------

def settings():
    """Environment, falling back to the settings file Ansible writes."""
    values = {}
    try:
        with open(SETTINGS_FILE) as f:
            for line in f:
                key, sep, value = line.strip().partition("=")
                if sep and not key.startswith("#"):
                    values[key] = value
    except OSError:
        pass
    values.update({k: v for k, v in os.environ.items() if v})
    return {
        "loki": values.get("LOKI_URL", "http://127.0.0.1:3100"),
        "prometheus": values.get("PROMETHEUS_URL", "http://127.0.0.1:9090"),
        "app_ssh_host": values.get("APP_SSH_HOST", "<app IP>"),
        "app_vpc_ip": values.get("APP_VPC_IP", "10.0.0.2"),
        "state_dir": values.get("STATE_DIRECTORY", "/var/lib/myguy-security"),
        "geoip_db": values.get("GEOIP_DB", "/var/lib/geoip/country.mmdb"),
        "telegram_token": values.get("TELEGRAM_BOT_TOKEN", ""),
        "telegram_chat": values.get("TELEGRAM_CHAT_ID", ""),
    }


# --- Reading Loki and Prometheus -------------------------------------------

class Sources:
    """Loki and Prometheus over HTTP.  Tests replace it with a fake."""

    def __init__(self, loki, prometheus):
        self.loki = loki.rstrip("/")
        self.prometheus = prometheus.rstrip("/")

    @staticmethod
    def _get(url, params):
        with urllib.request.urlopen(url + "?" + urllib.parse.urlencode(params), timeout=60) as r:
            body = json.load(r)
        if body.get("status") != "success":
            raise RuntimeError(f"{url}: {body}")
        return body["data"]

    def lines(self, query, start, end):
        """Every log line matching query in [start, end), oldest first, as
        (time in ns, stream labels, line)."""
        out, limit = [], 5000
        for _ in range(100):
            data = self._get(self.loki + "/loki/api/v1/query_range", {
                "query": query, "start": start, "end": end,
                "limit": limit, "direction": "forward"})
            page = sorted((int(ts), s["stream"], line)
                          for s in data["result"] for ts, line in s["values"])
            out.extend(page)
            if len(page) < limit:
                break
            start = page[-1][0] + 1
        return out

    def loki_vector(self, query, at):
        """A Loki metric query at one time, as [(labels, value)]."""
        data = self._get(self.loki + "/loki/api/v1/query", {"query": query, "time": at})
        return [(r["metric"], float(r["value"][1])) for r in data["result"]]

    def prom_vector(self, expr, at):
        data = self._get(self.prometheus + "/api/v1/query", {"query": expr, "time": at / 1e9})
        return [(r["metric"], float(r["value"][1])) for r in data["result"]]


def country_lookup(path):
    """An IP's country code from the DB-IP database (tasks/geoip.yml), looked
    up on this server; "" when the database or its reader isn't there."""
    try:
        import maxminddb  # python3-maxminddb (monitoring.yml)
        reader = maxminddb.open_database(path)
    except Exception:  # noqa: BLE001 - no country is fine
        return lambda ip: ""

    def country(ip):
        try:
            record = reader.get(ip) or {}
        except ValueError:
            return ""
        return (record.get("country") or {}).get("iso_code", "")
    return country


def no_country(ip):
    return ""


def place(settings_, ip):
    """The IP as untrusted code, with its country when known."""
    country = settings_.get("country", no_country)(ip)
    return code(ip, 45) + (f" ({esc(country)})" if country else "")


def top_countries(counter, n=3):
    # Most first; ties alphabetically, so the order is the same every day
    ranked = sorted(counter.items(), key=lambda x: (-x[1], x[0]))[:n]
    return ", ".join(f"{esc(c)} {k}" for c, k in ranked)


def total(vector):
    return sum(v for _, v in vector)


# --- Parsing ----------------------------------------------------------------

FAIL2BAN = re.compile(r"\[(?P<jail>[\w-]+)\]\s+(?P<action>Restore Ban|Ban|Unban)\s+(?P<ip>\S+)")
SSH_ACCEPTED = re.compile(r"Accepted (?P<method>\S+) for (?P<user>\S+) from (?P<ip>\S+) port \d+")
SSH_FAILED = re.compile(r"Invalid user|Failed \S+ for|\[preauth\]")
SSH_ADDRESS = re.compile(r"(?P<ip>\d{1,3}(?:\.\d{1,3}){3}|[0-9a-fA-F]*:[0-9a-fA-F:]+) port \d+")
SUDO = re.compile(r"^\s*(?P<user>\S+) : .*?COMMAND=(?P<command>.*)$")
# The backend's sign-in log (internal/api/handlers.go logAuthEvent)
AUTH = re.compile(r'auth event=(?P<event>\S+) email="(?P<email>[^"]*)" ip=(?P<ip>\S+)')
AUTH_QUERY = '{job="containers", service="api"} |= "auth event="'

DEPLOY = re.compile(r"^(?P<what>start|end) run=(?P<run>\S+)(?P<rest>.*)$")


def valid_ip(text):
    try:
        return str(ipaddress.ip_address(text))
    except ValueError:
        return None


def category(rule_id):
    return CATEGORIES.get(rule_id[:3], "other")


def parse_waf(ts, line):
    """One ModSecurity audit line: JSON, one per flagged request
    (coraza.conf.j2: {"transaction": {"client_ip", "request", "response",
    "messages"}})."""
    try:
        t = json.loads(line)["transaction"]
    except (ValueError, KeyError, TypeError):
        return None
    ip = valid_ip(str(t.get("client_ip", "")))
    if not ip:
        return None
    request = t.get("request") or {}
    headers = {str(k).lower(): v for k, v in (request.get("headers") or {}).items()}
    ids = [str((m.get("details") or {}).get("ruleId", "")) for m in t.get("messages") or []]
    findings = [i for i in ids if i and i[:3] not in EVALUATION]
    serious = sorted({category(i) for i in findings if i[:3] in SERIOUS})
    kinds = serious or sorted({category(i) for i in findings})
    return {
        "ts": ts,
        "ip": ip,
        "method": str(request.get("method") or "?"),
        # The path only: query strings can carry personal data
        "path": str(request.get("uri") or "?").split("?")[0],
        "status": int((t.get("response") or {}).get("http_code") or 0),
        "agent": str(headers.get("user-agent", "")).strip(),
        "flagged": FLAGGED_RULE in ids or bool(serious),
        "serious": bool(serious),
        "kinds": kinds or ["other"],
    }


def parse_fail2ban(ts, line):
    m = FAIL2BAN.search(line)
    ip = valid_ip(m.group("ip")) if m else None
    return {"ts": ts, "jail": m.group("jail"), "action": m.group("action"), "ip": ip} if ip else None


def parse_auth(ts, line):
    m = AUTH.search(line)
    ip = valid_ip(m.group("ip")) if m else None
    return {"ts": ts, "event": m.group("event"), "email": m.group("email"), "ip": ip} if ip else None


def auth_events(src, start, end, ip=None, email=None):
    query = AUTH_QUERY
    if ip:
        query += f' |= "ip={ip}"'
    if email:
        query += ' |= "email=\\"%s\\""' % email.replace('"', "")
    events = [e for e in (parse_auth(ts, l) for ts, _, l in src.lines(query, start, end)) if e]
    # |= matched a prefix (ip=1.2.3.4 in ip=1.2.3.45): keep exact ones
    return [e for e in events if (not ip or e["ip"] == ip) and (not email or e["email"] == email)]


def mask(email):
    """j***@gmail.com: enough to recognise, for Telegram."""
    local_part, _, domain = email.partition("@")
    return (local_part[:1] + "***@" + domain) if domain else "***"


def parse_falco(ts, line):
    try:
        event = json.loads(line)
    except ValueError:
        return None
    if not isinstance(event, dict) or "rule" not in event:
        return None
    fields = event.get("output_fields") or {}
    return {
        "ts": ts,
        "rule": event["rule"],
        "priority": event.get("priority", ""),
        "process": fields.get("proc.name") or "",
        "container": fields.get("container.name") or "host",
        "user": fields.get("user.name") or "",
    }


# --- Formatting -------------------------------------------------------------

def esc(value):
    return html.escape(str(value), quote=False)


def code(value, width=60):
    """Untrusted text: shortened, escaped, and never a link."""
    value = " ".join(str(value).split())
    if len(value) > width:
        value = value[:width - 1] + "…"
    return f"<code>{esc(value)}</code>"


def local(ts, fmt="%H:%M"):
    return dt.datetime.fromtimestamp(ts / 1e9, TZ).strftime(fmt)


def number(value):
    if value is None:
        return "—"
    if isinstance(value, float) and not value.is_integer():
        return f"{value:.1f}"
    return f"{int(value):,}"


def compare(current, previous):
    """ ↑ from 120 / ↓ from 120 / = against a previous value, if known."""
    if current is None or previous is None:
        return ""
    if round(current, 1) == round(previous, 1):
        return " ="
    return f" {'↑' if current > previous else '↓'} from {number(previous)}"


def ban_command(settings_, ip):
    return f"ssh ops@{settings_['app_ssh_host']} 'sudo fail2ban-client set manual banip {ip}'"


# --- Shared lookups ---------------------------------------------------------

def ban_states(src, now):
    """Per IP over Loki's 8 days: banned now (and by which jail), and how
    many bans it collected.  From fail2ban's own log."""
    events = [e for e in (parse_fail2ban(ts, l) for ts, _, l in
              src.lines('{job="fail2ban"} |~ "Ban|Unban"', now - 8 * DAY, now)) if e]
    states = defaultdict(lambda: {"jails": {}, "bans": 0, "ban_times": []})
    for e in events:
        state = states[e["ip"]]
        if e["action"] == "Unban":
            state["jails"].pop(e["jail"], None)
        else:
            state["jails"][e["jail"]] = e["ts"]
            if e["action"] == "Ban":
                state["bans"] += 1
                state["ban_times"].append(e["ts"])
    return states


def real_user(src, ip, now):
    """Signed in from this IP in the last 8 days: banning it would lock a
    real person out (mobile carriers share IPs between many people)."""
    query = ('sum(count_over_time({job="nginx_access"} |= "\\"remote_addr\\":\\"%s\\"" '
             '| json | uri="/api/v1/profile" | status="200" [8d]))' % ip)
    return total(src.loki_vector(query, now)) > 0


def deploy_windows(src, start, end):
    """[(start, end)] of deploys, from the workflow's markers."""
    runs = {}
    for ts, _, line in src.lines('{job="deploy"}', start - DEPLOY_LONGEST, end):
        m = DEPLOY.match(line.strip())
        if not m:
            continue
        run = runs.setdefault(m.group("run"), {"start": None, "end": None, "rest": ""})
        run[m.group("what")] = ts
        if m.group("what") == "start":
            run["rest"] = m.group("rest").strip()
    windows = []
    for run in runs.values():
        begin = run["start"] or (run["end"] - DEPLOY_LONGEST)
        windows.append((begin, run["end"] or begin + DEPLOY_LONGEST, run["rest"]))
    return sorted(windows)


def waf_entries(src, start, end, ip=None):
    query = '{job="modsecurity"}' + (f' |= "\\"client_ip\\":\\"{ip}\\""' if ip else "")
    return [d for d in (parse_waf(ts, l) for ts, _, l in src.lines(query, start, end)) if d]


# --- check: attacks ---------------------------------------------------------

def attack_activity(src, now):
    """Per IP over the last 10 minutes: WAF findings and sign-in calls."""
    activity = defaultdict(lambda: {"hits": 0, "serious": 0, "succeeded": 0, "first": None,
                                    "last": None, "kinds": Counter(), "paths": Counter(),
                                    "agent": "", "sign_in": 0})
    for d in waf_entries(src, now - WINDOW, now):
        if not d["flagged"]:
            continue
        a = activity[d["ip"]]
        a["hits"] += 1
        a["first"] = a["first"] or d["ts"]
        a["last"] = d["ts"]
        a["kinds"].update(d["kinds"])
        a["paths"][f"{d['method']} {d['path']}"] += 1
        a["agent"] = d["agent"] or a["agent"]
        if d["serious"]:
            a["serious"] += 1
            if d["status"] and d["status"] < 400:
                a["succeeded"] += 1
    sign_in = src.loki_vector('sum by (remote_addr) (count_over_time({job="nginx_access"} | json '
                              '| method="POST" | uri=~"/api/v1/auth/.*" [10m]))', now)
    for labels, value in sign_in:
        ip = valid_ip(labels.get("remote_addr", ""))
        if ip:
            activity[ip]["sign_in"] = int(value)
    return activity


def reasons_for(a, banned, now):
    """Why an IP needs you, if it does.  Banned attackers are left to
    fail2ban: they're in the daily summary, not a message."""
    reasons = []
    if a["succeeded"]:
        reasons.append("succeeded")
    if a["hits"] >= WAF_HITS_UNBANNED and not banned and a["first"] and now - a["first"] >= WAF_GRACE:
        reasons.append("not banned")
    if a["sign_in"] > SIGN_IN_CALLS and not banned:
        reasons.append("sign-in abuse")
    return reasons


REASON_TEXT = {
    "succeeded": "⚠️ serious attack requests answered without an error (not blocked: the WAF only logs)",
    "not banned": "fail2ban hasn't banned it (it should after 3 WAF hits)",
    "sign-in abuse": "sign-in calls far above a real sign-in (code guessing or email bombing)",
}


def describe_ip(src, settings_, ip, a, ban, now):
    jails = ban["jails"] if ban else {}
    banned = (f"banned by {esc(', '.join(sorted(jails)))} since {local(max(jails.values()))}"
              if jails else "not banned")
    before = (ban["bans"] - len([t for t in ban["ban_times"] if t >= (a["first"] or now)])) if ban else 0
    parts = [f"{place(settings_, ip)} · {a['hits']} flagged"
             + (f", {a['sign_in']} sign-in calls" if a["sign_in"] else "") + " in 10 min"]
    if a["kinds"]:
        parts[0] += " · " + esc(", ".join(k for k, _ in a["kinds"].most_common(2)))
    if a["paths"]:
        parts.append(f"  {code(a['paths'].most_common(1)[0][0], 50)}"
                     + (f" · agent {code(a['agent'], 40)}" if a["agent"] else ""))
    if a["succeeded"]:
        parts.append(f"  ⚠️ {a['succeeded']} serious request(s) answered without an error")
    if a["sign_in"]:
        try:
            emails = Counter(e["email"] for e in auth_events(src, now - WINDOW, now, ip=ip))
            if emails:
                parts.append(f"  tried {len(emails)} address(es): "
                             + ", ".join(code(mask(e), 40) for e, _ in emails.most_common(3)))
        except Exception:  # noqa: BLE001
            pass
    parts.append(f"  fail2ban: {banned}" + (f" · banned {before}× before (8 days)" if before > 0 else ""))
    if a["first"]:
        parts.append(f"  active {local(a['first'])}–{local(a['last'])}")
    try:
        parts[-1] += " · " + ("⚠️ someone signed in from this IP: may be shared"
                              if real_user(src, ip, now) else "no real-user activity")
    except Exception:  # noqa: BLE001 - one lookup failing shouldn't stop the alert
        pass
    return "\n".join(parts)


def attack_messages(src, settings_, state, now):
    activity = attack_activity(src, now)
    bans = ban_states(src, now)
    alerts = state.setdefault("attacks", {})
    to_send = []
    for ip, a in activity.items():
        ban = bans.get(ip)
        banned = bool(ban and ban["jails"])
        reasons = reasons_for(a, banned, now)
        known = alerts.get(ip)
        if known:
            known["last"] = max(known["last"], a["last"] or now)
        if not reasons:
            continue
        new = [r for r in reasons if r not in (known or {}).get("reasons", [])]
        if not known:
            known = alerts[ip] = {"first": a["first"] or now, "last": a["last"] or now,
                                  "reasons": [], "alerted": now}
        known["reasons"] = sorted(set(known["reasons"]) | set(reasons))
        if new:
            to_send.append((ip, a, ban, new))
            # In Loki (job="security-checks"), for the Defences dashboard
            print(f"alert: attack ip={ip} why={','.join(new).replace(' ', '-')}", flush=True)
    messages = []
    if to_send:
        to_send.sort(key=lambda x: (-x[1]["succeeded"], -x[1]["hits"], -x[1]["sign_in"]))
        lines = [f"🚨 <b>Attack needs a look</b> ({len(to_send)} IP{'s' if len(to_send) > 1 else ''})"]
        for ip, a, ban, new in to_send[:TOP_IPS]:
            lines.append("")
            lines.append(describe_ip(src, settings_, ip, a, ban, now))
            lines.append("  why: " + "; ".join(REASON_TEXT[r] for r in new))
        if len(to_send) > TOP_IPS:
            lines.append(f"\n+{len(to_send) - TOP_IPS} more: Defences dashboard")
        first_ip = to_send[0][0]
        lines.append(f"\nNext: Investigate IP dashboard (ip = {code(first_ip, 45)}), or on the "
                     f"monitoring server: {code('investigate ' + first_ip, 60)}")
        lines.append(f"Ban 7 days: {code(ban_command(settings_, first_ip), 120)}")
        messages.append("\n".join(lines))
        state.setdefault("alert_log", []).extend([now] * len(to_send))
    # Attacks that went quiet: one line each on how they ended
    ended = []
    for ip, known in list(alerts.items()):
        if ip in activity or now - known["last"] < EPISODE_GAP:
            continue
        ban = bans.get(ip)
        try:
            hits = len([d for d in waf_entries(src, known["first"], now, ip) if d["flagged"]])
        except Exception:  # noqa: BLE001
            hits = None
        banned = (f"banned ({esc(', '.join(sorted(ban['jails'])))})" if ban and ban["jails"]
                  else "not banned")
        ended.append(f"{place(settings_, ip)} · {local(known['first'])}–{local(known['last'])} · "
                     f"{number(hits)} flagged in total · {banned}")
        del alerts[ip]
    if ended:
        messages.append("✅ <b>Attack ended</b>\n" + "\n".join(ended))
    return messages


# --- check: SSH logins ------------------------------------------------------

def ssh_messages(src, settings_, state, now):
    """Logins nobody expected:
      myguy (the CI key) outside a deploy, on either server;
      on the monitoring server, anything not from the app server (ProxyJump);
      ops on the app server from an IP not seen in the last 8 days;
      any other user."""
    since = state.get("ssh_since", now - WINDOW)
    seen = {k: t for k, t in state.get("ssh_seen", {}).items() if now - t < HOUR}
    known_ips = {ip: t for ip, t in state.get("ops_ips", {}).items() if now - t < 8 * DAY}
    windows = deploy_windows(src, since - 2 * MIN, now)
    alerts = []
    for ts, labels, line in src.lines('{job="ssh"} |= "Accepted"', since - 2 * MIN, now):
        m = SSH_ACCEPTED.search(line)
        key = f"{ts}:{labels.get('host')}:{line[:80]}"
        if not m or key in seen:
            continue
        seen[key] = ts
        host, user, ip = labels.get("host", "?"), m.group("user"), m.group("ip")
        in_deploy = any(begin - 2 * MIN <= ts <= end + 5 * MIN for begin, end, _ in windows)
        why = None
        if user == "myguy":
            if not in_deploy:
                why = ("the CI key (myguy) logged in while no deploy was running. If you didn't "
                       "run one by hand, rotate SSH_PRIVATE_KEY and check sudo use")
            elif host == "monitoring" and ip != settings_["app_vpc_ip"]:
                why = "the monitoring server should only be reached through the app server"
        elif host == "monitoring":
            if ip != settings_["app_vpc_ip"]:
                why = "the monitoring server should only be reached through the app server"
        elif user == "ops":
            if ip not in known_ips:
                why = ("ops logged in from an IP not seen in the last 8 days. If it wasn't you, "
                       "replace OPS_SSH_PUBLIC_KEY and check sudo use")
            known_ips[ip] = ts
        else:
            why = f"no one but ops and myguy should log in (user {user})"
        if why:
            print(f"alert: ssh host={host} user={user} ip={ip}", flush=True)
            alerts.append(f"{esc(host)} · user {code(user, 30)} · from {place(settings_, ip)} · "
                          f"{local(ts)}\nwhy: {esc(why)}")
    state["ssh_since"], state["ssh_seen"], state["ops_ips"] = now, seen, known_ips
    if not alerts:
        return []
    state.setdefault("ssh_alert_log", []).extend([now] * len(alerts))
    return ["🔐 <b>Unexpected SSH login</b>\n" + "\n\n".join(alerts)
            + "\n\nNext: Defences dashboard (SSH logins, sudo)"]


# --- check: bans that may have hit a real person --------------------------------

# Jails that ban automatically for web traffic.  Not sshd (people don't SSH)
# and not manual (you banned it yourself).
AUTO_WEB_JAILS = {"nginx-4xx", "nginx-botsearch", "nginx-modsecurity", "nginx-auth-abuse", "tripwire"}


def ban_messages(src, settings_, state, now):
    """A new automatic ban of an IP someone signed in from in the last 8
    days: probably shared (a mobile carrier IP, an office), so a real
    person may now be locked out.  That's a decision: unban or not."""
    since = state.get("bans_since", now - WINDOW)
    seen = {k: t for k, t in state.get("bans_seen", {}).items() if now - t < HOUR}
    alerts = []
    for ts, _, line in src.lines('{job="fail2ban"} |= "] Ban "', since - 2 * MIN, now):
        e = parse_fail2ban(ts, line)
        key = f"{ts}:{line[-80:]}"
        if not e or e["jail"] not in AUTO_WEB_JAILS or key in seen:
            continue
        seen[key] = ts
        if not real_user(src, e["ip"], now):
            continue
        # What got it banned: its last requests before the ban
        last = []
        for _, _, l in src.lines('{job="nginx_access"} |= "\\"remote_addr\\":\\"%s\\""' % e["ip"], ts - WINDOW, ts + MIN):
            try:
                req = json.loads(l)
            except ValueError:
                continue
            last.append(f"{req.get('method', '')} {req.get('uri', '')} {req.get('status', '')}")
        print(f"alert: banned-user ip={e['ip']} jail={e['jail']}", flush=True)
        alerts.append(
            f"{place(settings_, e['ip'])} · banned by {esc(e['jail'])} at {local(ts)}"
            + (f" · after {code(last[-1], 50)}" if last else "")
            + "\nSomeone signed in from this IP in the last 8 days: it may be shared, so a real person may be locked out."
            + f"\nNext: {code('investigate ' + e['ip'], 60)}. If it's a real user, unban: "
            + code(f"ssh ops@{settings_['app_ssh_host']} 'sudo fail2ban-client set {e['jail']} unbanip {e['ip']}'", 140))
    state["bans_since"], state["bans_seen"] = now, seen
    if not alerts:
        return []
    return [f"⚠️ <b>Ban may have hit a real user</b> ({len(alerts)})\n\n" + "\n\n".join(alerts)]


def check(src, settings_, state, now, send):
    messages = (attack_messages(src, settings_, state, now) + ssh_messages(src, settings_, state, now)
                + ban_messages(src, settings_, state, now))
    for message in messages:
        send(message)
    for log in ("alert_log", "ssh_alert_log"):
        state[log] = [t for t in state.get(log, []) if now - t < HISTORY_DAYS * DAY]
    print(f"check ok: {len(messages)} message(s)", flush=True)


# --- Summaries --------------------------------------------------------------

VISITOR_QUERY = ('count by (remote_addr) (count_over_time({job="nginx_access"} | json '
                 '| uri =~ "/assets/.*[.]js|/(api/v1|store/api/v1|chat/api/v1)/.*" | status < 400 '
                 '| user_agent !~ "(?i).*(bot|crawl|spider|slurp|curl|wget|python|go-http|monitor|uptime|headless).*" '
                 '[%s]))')


def collect(src, settings_, state, start, end):
    """The day's numbers.  A section that can't be read is named in
    "unreadable" (and in Needs attention) rather than stopping the rest."""
    s = {"unreadable": []}
    country = settings_.get("country", no_country)
    attackers = set()  # IPs flagged by the WAF, banned, or failing SSH

    def section(name, fn):
        try:
            fn()
        except Exception as e:  # noqa: BLE001
            print(f"summary: couldn't read {name}: {e}", file=sys.stderr)
            s["unreadable"].append(name)

    span = f"{(end - start) // HOUR}h"

    def health():
        site = src.prom_vector(f'min(avg_over_time(probe_success{{job="site"}}[{span}]))', end)
        s["uptime"] = round(100 * total(site), 2) if site else None
        # Failed probes at the prober's 15 s interval (outages are often
        # shorter than a minute), as minutes
        down = src.prom_vector(f'sum_over_time((1 - min(probe_success{{job="site"}}))[{span}:15s]) / 4', end)
        s["outage_minutes"] = round(total(down))
        cert = src.prom_vector('min((probe_ssl_earliest_cert_expiry{job="site"} - time()) / 86400)', end)
        s["cert_days"] = int(total(cert)) if cert else None
        disk = src.prom_vector('max by (instance) (100 * (1 - node_filesystem_avail_bytes{mountpoint="/"} '
                               '/ node_filesystem_size_bytes{mountpoint="/"}))', end)
        s["disk"] = {m.get("instance", "?"): round(v) for m, v in disk}
        memory = src.prom_vector('max by (instance) (100 * max_over_time((1 - node_memory_MemAvailable_bytes '
                                 f'/ node_memory_MemTotal_bytes)[{span}:5m]))', end)
        s["memory_peak"] = {m.get("instance", "?"): round(v) for m, v in memory}

    def errors():
        five = f'count_over_time({{job="nginx_access"}} | json | status >= 500 [{span}])'
        s["server_errors"] = int(total(src.loki_vector(f"sum({five})", end)))
        top = src.loki_vector(f"topk(1, sum by (method, uri) ({five}))", end)
        s["top_error"] = f"{top[0][0].get('method', '')} {top[0][0].get('uri', '')}".strip() if top else ""

    def deploys():
        windows = [w for w in deploy_windows(src, start, end) if start <= w[0] < end]
        s["deploys"] = [[local(b), rest] for b, _, rest in windows]

    def defences():
        entries = [d for d in waf_entries(src, start, end) if d["flagged"]]
        s["waf"] = len(entries)
        s["waf_ips"] = len({d["ip"] for d in entries})
        s["waf_succeeded"] = sum(1 for d in entries if d["serious"] and 0 < d["status"] < 400)
        s["kinds"] = dict(Counter(d["kinds"][0] for d in entries))
        bans = [e for e in (parse_fail2ban(ts, l) for ts, _, l in
                src.lines('{job="fail2ban"} |~ "Ban|Unban"', start, end)) if e]
        s["bans"] = dict(Counter(e["jail"] for e in bans if e["action"] == "Ban"))
        s["unbans"] = sum(1 for e in bans if e["action"] == "Unban")
        banned = {e["ip"] for e in bans if e["action"] == "Ban"}
        s["banned_ips"] = sorted(banned)
        by_ip = defaultdict(Counter)
        for d in entries:
            by_ip[d["ip"]].update(d["kinds"][:1])
        top = sorted(by_ip.items(), key=lambda x: -sum(x[1].values()))[:3]
        s["top_ips"] = [[ip, sum(c.values()), c.most_common(1)[0][0], ip in banned, country(ip)]
                        for ip, c in top]
        attackers.update(by_ip, banned)
        auth = f'{{job="nginx_access"}} | json | uri=~"/api/v1/auth/.*"'
        s["code_requests"] = int(total(src.loki_vector(
            f'sum(count_over_time({auth} | method="POST" | uri="/api/v1/auth/request-code" [{span}]))', end)))
        s["auth_429"] = int(total(src.loki_vector(f'sum(count_over_time({auth} | status=429 [{span}]))', end)))
        # Requests nginx refused because fail2ban banned the IP
        s["refused"] = int(total(src.loki_vector(
            f'sum(count_over_time({{job="nginx_access"}} | json | status="444" [{span}]))', end)))

    def sign_in():
        events = auth_events(src, start, end)
        kinds = Counter(e["event"] for e in events)
        sent = [e for e in events if e["event"] in ("code_sent", "limited")]
        s["sign_in"] = {k: kinds.get(k, 0) for k in ("code_sent", "limited", "wrong_code", "signed_in", "new_account", "signed_up")}
        s["sign_in_addresses"] = len({e["email"] for e in sent})
        s["sign_in_ips"] = len({e["ip"] for e in sent})
        top = Counter(e["email"] for e in sent).most_common(1)
        # Only worth naming when far above a real sign-in (1-2 codes)
        s["most_targeted"] = [mask(top[0][0]), top[0][1]] if top and top[0][1] >= 5 else None

    def access():
        logins, failed_ips, failed = Counter(), set(), 0
        for _, labels, line in src.lines('{job="ssh"}', start, end):
            m = SSH_ACCEPTED.search(line)
            if m:
                logins[f"{m.group('user')}@{labels.get('host', '?')}"] += 1
            elif SSH_FAILED.search(line):
                failed += 1
                a = SSH_ADDRESS.search(line)
                if a:
                    failed_ips.add(a.group("ip"))
        s["ssh_logins"] = dict(logins)
        s["ssh_failed"] = failed
        s["ssh_failed_ips"] = len(failed_ips)
        attackers.update(failed_ips)
        s["unknown_logins"] = len([t for t in state.get("ssh_alert_log", []) if start <= t < end])
        commands = Counter()
        for _, labels, line in src.lines('{job="sudo"} |= "COMMAND="', start, end):
            m = SUDO.match(line)
            if m and m.group("user") == "ops":
                commands[f"{labels.get('host', '?')}: {m.group('command').strip()}"] += 1
        s["ops_sudo"] = sum(commands.values())
        s["ops_commands"] = [c for c, _ in commands.most_common(3)]

    def runtime():
        events = [e for e in (parse_falco(ts, l) for ts, _, l in src.lines('{job="falco"}', start, end)) if e]
        s["falco_critical"] = sum(1 for e in events if e["priority"] in FALCO_CRITICAL)
        warnings = Counter(e["rule"] for e in events if e["priority"] == "Warning")
        s["falco_warnings"] = sum(warnings.values())
        s["falco_rules"] = [r for r, _ in warnings.most_common(2)]

    def visitors():
        ips = [m.get("remote_addr", "") for m, _ in src.loki_vector(VISITOR_QUERY % span, end)]
        s["visitors"] = len(ips)
        s["visitor_countries"] = dict(Counter(c for c in map(country, ips) if c))
        # Growth over the window (works while Prometheus has less history)
        accounts = src.prom_vector(f"max(myguy_accounts) - min(min_over_time(myguy_accounts[{span}]))", end)
        s["signups"] = max(0, int(total(accounts))) if accounts else None

    for name, fn in (("health", health), ("server errors", errors), ("deploys", deploys),
                     ("defences", defences), ("sign-in", sign_in), ("access", access), ("Falco", runtime),
                     ("visitors", visitors)):
        section(name, fn)
    # Distinct IPs per country (an IP counts once however much it did)
    s["attacker_countries"] = dict(Counter(c for c in map(country, attackers) if c))
    s["attack_alerts"] = len([t for t in state.get("alert_log", []) if start <= t < end])
    return s


def attention(s):
    """The short list at the bottom: anything you should look at."""
    items = []
    if s.get("unknown_logins"):
        items.append(f"{s['unknown_logins']} unexpected SSH login(s)")
    if s.get("waf_succeeded"):
        items.append(f"{s['waf_succeeded']} serious attack request(s) answered without an error")
    if s.get("attack_alerts"):
        items.append(f"{s['attack_alerts']} attacking IP(s) needed a look (see the alerts)")
    if s.get("falco_critical"):
        items.append(f"{s['falco_critical']} critical Falco event(s)")
    # A minute or two is a deploy restarting things; Site down fires at 3
    if (s.get("outage_minutes") or 0) >= 3:
        items.append(f"site down ~{s['outage_minutes']} min")
    if s.get("cert_days") is not None and s["cert_days"] < 14:
        items.append(f"certificate expires in {s['cert_days']} days")
    for host, used in (s.get("disk") or {}).items():
        if used >= 80:
            items.append(f"{host} disk {used}%")
    for host, used in (s.get("memory_peak") or {}).items():
        if used >= 90:
            items.append(f"{host} memory peaked at {used}%")
    if (s.get("server_errors") or 0) > 20:
        items.append(f"{s['server_errors']} server errors")
    if s.get("unreadable"):
        items.append("couldn't read: " + ", ".join(s["unreadable"]))
    return items


def format_summary(title, s, prev, days=1):
    """Daily (days=1) or weekly summary text, comparing with prev."""
    p = prev or {}
    per = "in 24 h" if days == 1 else "this week"

    def by_host(values, unit="%"):
        return " / ".join(f"{h} {v}{unit}" for h, v in sorted((values or {}).items())) or "—"

    lines = [f"🛡 <b>{esc(title)}</b>", "", "<b>Health</b>"]
    uptime = s.get("uptime")
    lines.append(f"  Site up {number(uptime)}%" + (f" (down ~{s['outage_minutes']} min)"
                 if s.get("outage_minutes") else "")
                 + (f" · certificate valid {s['cert_days']} days" if s.get("cert_days") is not None else ""))
    lines.append(f"  Disk {by_host(s.get('disk'))} · memory peak {by_host(s.get('memory_peak'))}")
    lines.append(f"  Server errors: {number(s.get('server_errors'))}{compare(s.get('server_errors'), p.get('server_errors'))}"
                 + (f" (top {code(s['top_error'], 40)})" if s.get("top_error") else ""))
    deploys = s.get("deploys") or []
    if days == 1:
        lines.append(f"  Deploys: {len(deploys)}" + (" · " + ", ".join(
            f"{esc(t)} ({esc(rest.replace('scope=', '').replace('services=', ''))})" for t, rest in deploys[:3])
            if deploys else ""))
    else:
        lines.append(f"  Deploys: {s.get('deploy_count', 0)}{compare(s.get('deploy_count'), p.get('deploy_count'))}")

    lines += ["", "<b>Defences</b>"]
    lines.append(f"  WAF detections: {number(s.get('waf'))}{compare(s.get('waf'), p.get('waf'))}"
                 + (f" from {number(s['waf_ips'])} IPs" if s.get("waf_ips") is not None else "")
                 + f" · succeeded: {number(s.get('waf_succeeded'))}")
    bans = s.get("bans") or {}
    lines.append(f"  Bans: {sum(bans.values())}{compare(sum(bans.values()), sum((p.get('bans') or {}).values()) if prev else None)}"
                 + (" (" + ", ".join(f"{esc(j)} {n}" for j, n in sorted(bans.items(), key=lambda x: -x[1])) + ")"
                    if bans else "") + f" · unbans: {number(s.get('unbans'))}")
    if s.get("top_ips"):
        lines.append("  Top IPs:")
        for ip, hits, kind, banned, *rest in s["top_ips"]:
            where = f" ({esc(rest[0])})" if rest and rest[0] else ""
            lines.append(f"    {code(ip, 45)}{where} {hits} hits · {esc(kind)} · {'banned' if banned else 'not banned'}")
    if days > 1:
        if s.get("repeat_offenders"):
            lines.append(f"  Repeat offenders: {len(s['repeat_offenders'])} IPs banned on 2+ days")
        if s.get("busiest_day"):
            lines.append(f"  Busiest day: {esc(s['busiest_day'])}")
        kinds = s.get("kinds") or {}
        if kinds:
            all_ = sum(kinds.values())
            lines.append("  Attack types: " + ", ".join(
                f"{esc(k)} {round(100 * n / all_)}%" for k, n in sorted(kinds.items(), key=lambda x: -x[1])[:3]))
    if s.get("attacker_countries"):
        lines.append("  Attacker countries (IPs): " + top_countries(Counter(s["attacker_countries"]), 5))
    if s.get("refused") is not None:
        lines.append(f"  Requests refused from banned IPs: {number(s.get('refused'))}"
                     f"{compare(s.get('refused'), p.get('refused'))}")
    lines.append(f"  Sign-in: {number(s.get('code_requests'))} code requests"
                 f"{compare(s.get('code_requests'), p.get('code_requests'))} · {number(s.get('auth_429'))} rate-limited (429)")
    si = s.get("sign_in") or {}
    if si:
        lines.append(f"    codes sent to {number(s.get('sign_in_addresses'))} address(es) from {number(s.get('sign_in_ips'))} IP(s)"
                     f" · {number(si.get('signed_in', 0) + si.get('signed_up', 0))} signed in"
                     f" ({number(si.get('signed_up'))} new) · {number(si.get('wrong_code'))} wrong codes")
    if s.get("most_targeted"):
        lines.append(f"    most codes: {code(s['most_targeted'][0], 40)} ×{s['most_targeted'][1]}"
                     " (many from several IPs: someone may be email-bombing it)")

    lines += ["", "<b>Access</b>"]
    logins = s.get("ssh_logins") or {}
    lines.append("  SSH logins: " + (", ".join(f"{esc(k)} ×{v}" for k, v in sorted(logins.items())) or "none")
                 + f" · unexpected: {number(s.get('unknown_logins'))}")
    lines.append(f"  Failed SSH attempts: {number(s.get('ssh_failed'))}{compare(s.get('ssh_failed'), p.get('ssh_failed'))}"
                 + (f" from {s['ssh_failed_ips']} IPs" if s.get("ssh_failed_ips") else ""))
    lines.append(f"  sudo by ops: {number(s.get('ops_sudo'))}"
                 + (" (" + ", ".join(code(c, 60) for c in s.get("ops_commands") or []) + ")"
                    if s.get("ops_commands") else ""))

    lines += ["", "<b>Runtime (Falco)</b>"]
    lines.append(f"  Critical: {number(s.get('falco_critical'))} · warnings: {number(s.get('falco_warnings'))}"
                 f"{compare(s.get('falco_warnings'), p.get('falco_warnings'))}"
                 + (" (" + ", ".join(code(r, 40) for r in s.get("falco_rules") or []) + ")"
                    if s.get("falco_rules") else ""))

    lines += ["", "<b>Visitors</b>"]
    lines.append(f"  ~{number(s.get('visitors'))} real visitors{compare(s.get('visitors'), p.get('visitors'))}"
                 f" · sign-ups {number(s.get('signups'))}{compare(s.get('signups'), p.get('signups'))}")
    if s.get("visitor_countries"):
        lines.append("  From: " + top_countries(Counter(s["visitor_countries"]), 5))

    items = attention(s)
    lines += ["", "<b>Needs attention:</b> " + ("nothing ✅" if not items else "")]
    lines += [f"  • {esc(i)}" for i in items]
    if not prev:
        lines.append(f"\n(Comparisons start {'tomorrow' if days == 1 else 'next week'}.)")
    return "\n".join(lines)


def load_history(settings_):
    path = os.path.join(settings_["state_dir"], "history.jsonl")
    records = []
    try:
        with open(path) as f:
            records = [json.loads(line) for line in f if line.strip()]
    except OSError:
        pass
    return records


def save_history(settings_, records, today):
    keep = [r for r in records if (today - dt.date.fromisoformat(r["date"])).days < HISTORY_DAYS]
    path = os.path.join(settings_["state_dir"], "history.jsonl")
    with open(path + ".tmp", "w") as f:
        for r in keep:
            f.write(json.dumps(r) + "\n")
    os.replace(path + ".tmp", path)


def find(records, kind, date):
    for r in records:
        if r["kind"] == kind and r["date"] == date.isoformat():
            return r["stats"]
    return None


def daily(src, settings_, state, now, send, persist=True):
    today = dt.datetime.fromtimestamp(now / 1e9, TZ).date()
    stats = collect(src, settings_, state, now - DAY, now)
    records = [r for r in load_history(settings_) if not (r["kind"] == "daily" and r["date"] == today.isoformat())]
    prev = find(records, "daily", today - dt.timedelta(days=1))
    send(format_summary(f"MyGuy daily summary: {today:%a %d %b} (last 24 h)", stats, prev))
    if persist:
        records.append({"kind": "daily", "date": today.isoformat(), "stats": stats})
        save_history(settings_, records, today)


def weekly(src, settings_, state, now, send, persist=True):
    """The last 7 days, added up from the daily summaries (Loki keeps only
    8 days, so the week before can't be queried again)."""
    today = dt.datetime.fromtimestamp(now / 1e9, TZ).date()
    records = load_history(settings_)
    days = [find(records, "daily", today - dt.timedelta(days=n)) for n in range(7)]
    have = [(today - dt.timedelta(days=n), d) for n, d in enumerate(days) if d]
    w = {"unreadable": []}
    if len(have) < 7:
        w["unreadable"].append(f"{7 - len(have)} day(s) with no daily summary")
    for key in ("server_errors", "waf", "waf_succeeded", "unbans", "code_requests", "auth_429", "refused",
                "ssh_failed", "unknown_logins", "ops_sudo", "falco_critical", "falco_warnings",
                "signups", "outage_minutes", "attack_alerts"):
        w[key] = sum(d.get(key) or 0 for _, d in have)
    uptimes = [d["uptime"] for _, d in have if d.get("uptime") is not None]
    w["uptime"] = round(sum(uptimes) / len(uptimes), 2) if uptimes else None
    w["deploy_count"] = sum(len(d.get("deploys") or []) for _, d in have)
    w["bans"], w["kinds"], w["ssh_logins"] = Counter(), Counter(), Counter()
    banned_days = Counter()
    for _, d in have:
        w["bans"].update(d.get("bans") or {})
        w["kinds"].update(d.get("kinds") or {})
        w["ssh_logins"].update(d.get("ssh_logins") or {})
        banned_days.update(set(d.get("banned_ips") or []))
    w["repeat_offenders"] = sorted(ip for ip, n in banned_days.items() if n >= 2)
    if have:
        busiest_date, busiest = max(have, key=lambda x: x[1].get("waf") or 0)
        if busiest.get("waf"):
            w["busiest_day"] = f"{busiest_date:%a} ({busiest['waf']} detections)"
        latest = have[0][1]
        for key in ("disk", "cert_days"):
            w[key] = latest.get(key)
        w["memory_peak"] = {}
        for _, d in have:
            for h, v in (d.get("memory_peak") or {}).items():
                w["memory_peak"][h] = max(v, w["memory_peak"].get(h, 0))
    top = Counter()
    for _, d in have:
        for ip, hits, kind, banned, *rest in d.get("top_ips") or []:
            top[(ip, kind)] += hits
    country = settings_.get("country", no_country)
    w["top_ips"] = [[ip, hits, kind, ip in banned_days, country(ip)] for (ip, kind), hits in top.most_common(3)]
    w["attacker_countries"] = Counter()
    for _, d in have:
        w["attacker_countries"].update(d.get("attacker_countries") or {})
    w["attacker_countries"] = dict(w["attacker_countries"])
    try:
        ips = [m.get("remote_addr", "") for m, _ in src.loki_vector(VISITOR_QUERY % "7d", now)]
        w["visitors"] = len(ips)
        w["visitor_countries"] = dict(Counter(c for c in map(country, ips) if c))
    except Exception:  # noqa: BLE001
        w["unreadable"].append("visitors")
    w["bans"], w["kinds"], w["ssh_logins"] = dict(w["bans"]), dict(w["kinds"]), dict(w["ssh_logins"])
    prev = find(records, "weekly", today - dt.timedelta(days=7))
    start = today - dt.timedelta(days=6)
    send(format_summary(f"MyGuy weekly summary: {start:%a %d %b} – {today:%a %d %b} (vs the week before)",
                        w, prev, days=7))
    if persist:
        records = [r for r in records if not (r["kind"] == "weekly" and r["date"] == today.isoformat())]
        records.append({"kind": "weekly", "date": today.isoformat(), "stats": w})
        save_history(settings_, records, today)


# --- investigate --------------------------------------------------------------

def investigate(src, settings_, ip, hours, now, out=print):
    """Everything one IP did, as plain text for a terminal."""
    start = now - hours * HOUR
    country = settings_.get("country", no_country)(ip)
    out(f"Investigate {ip}" + (f" ({country})" if country else "") + f" (last {hours} h, times Europe/Amsterdam)")
    out("")
    requests = []
    for ts, _, line in src.lines('{job="nginx_access"} |= "\\"remote_addr\\":\\"%s\\""' % ip, start, now):
        try:
            requests.append((ts, json.loads(line)))
        except ValueError:
            continue
    if requests:
        statuses = Counter(f"{r['status'] // 100}xx" for _, r in requests if isinstance(r.get("status"), int))
        out(f"Requests: {len(requests)} ({', '.join(f'{k} {v}' for k, v in sorted(statuses.items()))}) · "
            f"first {local(requests[0][0], '%a %H:%M')}, last {local(requests[-1][0], '%a %H:%M')}")
        out("Top paths:")
        for path, n in Counter(f"{r.get('method')} {r.get('uri')} {r.get('status')}" for _, r in requests).most_common(10):
            out(f"  {n:>5}  {path[:100]}")
        out("User agents:")
        for agent, n in Counter(r.get("user_agent", "") for _, r in requests).most_common(5):
            out(f"  {n:>5}  {agent[:100]}")
        auth = [r for _, r in requests if str(r.get("uri", "")).startswith("/api/v1/auth/")]
        out(f"Sign-in calls: {len(auth)} · rate-limited (429): {sum(1 for r in auth if r.get('status') == 429)}")
    else:
        out("Requests: none")
    tried = defaultdict(Counter)
    try:
        for e in auth_events(src, start, now, ip=ip):
            tried[e["email"]][e["event"]] += 1
    except Exception as e:  # noqa: BLE001
        out(f"Sign-in addresses: couldn't read ({e})")
    if tried:
        out(f"Sign-in addresses tried from this IP: {len(tried)}")
        for email, events in sorted(tried.items(), key=lambda x: -sum(x[1].values()))[:15]:
            out(f"  {email:40} " + ", ".join(f"{k} {n}" for k, n in events.most_common()))
    try:
        out("Real user: " + ("signed in from this IP (8 days): may be shared, careful with bans"
                             if real_user(src, ip, now) else "no sign-ins from this IP (8 days)"))
    except Exception as e:  # noqa: BLE001
        out(f"Real user: couldn't check ({e})")
    out("")
    entries = waf_entries(src, start, now, ip)
    flagged = [d for d in entries if d["flagged"]]
    kinds = Counter(k for d in flagged for k in d["kinds"][:1])
    succeeded = [d for d in flagged if d["serious"] and 0 < d["status"] < 400]
    out(f"WAF: {len(flagged)} flagged" + (f" ({', '.join(f'{k} {n}' for k, n in kinds.most_common())})" if kinds else "") + " · "
        f"serious answered without an error: {len(succeeded)}")
    for d in flagged[-10:]:
        out(f"  {local(d['ts'], '%a %H:%M')}  {d['method']} {d['path'][:70]}  {d['status']}  {', '.join(d['kinds'])}")
    out("")
    ban = ban_states(src, now).get(ip)
    if ban and ban["jails"]:
        out("fail2ban: banned now by " + ", ".join(f"{j} (since {local(t, '%a %H:%M')})" for j, t in ban["jails"].items()))
    else:
        out("fail2ban: not banned now")
    out(f"Bans in the last 8 days: {ban['bans'] if ban else 0}")
    for ts, _, line in src.lines('{job="fail2ban"} |= " %s"' % ip, now - 8 * DAY, now)[-10:]:
        e = parse_fail2ban(ts, line)
        if e and e["ip"] == ip:
            out(f"  {local(ts, '%a %d %b %H:%M')}  {e['action']} ({e['jail']})")
    out("")
    ssh = src.lines('{job="ssh"} |= "%s"' % ip, start, now)
    accepted = [l for _, _, l in ssh if SSH_ACCEPTED.search(l)]
    out(f"SSH: {len(ssh) - len(accepted)} other lines, {len(accepted)} logins")
    for line in accepted[-5:]:
        out(f"  {line.strip()[:120]}")
    out("")
    out(f"Ban for 7 days: {ban_command(settings_, ip)}")
    out(f"Grafana: Investigate IP dashboard, ip = {ip}")


def investigate_email(src, settings_, email, hours, now, out=print):
    """Every sign-in step for one address: which IPs, how many codes."""
    events = auth_events(src, now - hours * HOUR, now, email=email)
    out(f"Sign-in for {email} (last {hours} h, times Europe/Amsterdam)")
    out("")
    if not events:
        out("No sign-in steps for this address.")
        return
    kinds = Counter(e["event"] for e in events)
    out("Steps: " + ", ".join(f"{k} {n}" for k, n in kinds.most_common())
        + f" · first {local(events[0]['ts'], '%a %H:%M')}, last {local(events[-1]['ts'], '%a %H:%M')}")
    country = settings_.get("country", no_country)
    ips = Counter(e["ip"] for e in events)
    out(f"From {len(ips)} IP(s):")
    for ip, n in ips.most_common(15):
        out(f"  {ip:40} {country(ip) or '--':3} {n}")
    if len(ips) >= 3 and kinds.get("code_sent", 0) + kinds.get("limited", 0) >= 5:
        out("")
        out("Many codes from several IPs: someone may be email-bombing this address.")
    out("")
    out(f"An IP in detail: investigate <IP>")


# --- Main ---------------------------------------------------------------------

def load_state(settings_):
    try:
        with open(os.path.join(settings_["state_dir"], "state.json")) as f:
            return json.load(f)
    except (OSError, ValueError):
        return {}


def save_state(settings_, state):
    path = os.path.join(settings_["state_dir"], "state.json")
    with open(path + ".tmp", "w") as f:
        json.dump(state, f)
    os.replace(path + ".tmp", path)


def telegram(settings_, dry_run):
    def send(text):
        if len(text) > 4000:
            text = text[:text.rfind("\n", 0, 3900)] + "\n… (cut short: see the dashboards)"
        if dry_run or not (settings_["telegram_token"] and settings_["telegram_chat"]):
            print(text, flush=True)
            if not dry_run:
                print("(Telegram isn't configured: printed instead)", file=sys.stderr)
            return
        data = urllib.parse.urlencode({
            "chat_id": settings_["telegram_chat"], "text": text,
            "parse_mode": "HTML", "disable_web_page_preview": "true"}).encode()
        url = f"https://api.telegram.org/bot{settings_['telegram_token']}/sendMessage"
        with urllib.request.urlopen(url, data=data, timeout=30) as r:
            r.read()
    return send


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("--dry-run", action="store_true", help="print messages instead of sending them")
    sub = parser.add_subparsers(dest="command", required=True)
    for name in ("check", "summary", "daily", "weekly"):
        sub.add_parser(name)
    inv = sub.add_parser("investigate", help="everything one IP did, or every sign-in step for an email")
    inv.add_argument("ip", metavar="IP_OR_EMAIL")
    inv.add_argument("--hours", type=int, default=24, help="how far back (at most 192, Loki's 8 days)")
    args = parser.parse_args(argv)

    settings_ = settings()
    settings_["country"] = country_lookup(settings_["geoip_db"])
    src = Sources(settings_["loki"], settings_["prometheus"])
    now = int(dt.datetime.now(dt.timezone.utc).timestamp()) * 10**9

    if args.command == "investigate":
        hours = max(1, min(args.hours, 192))
        if "@" in args.ip:
            investigate_email(src, settings_, args.ip.strip().lower(), hours, now)
            return 0
        ip = valid_ip(args.ip)
        if not ip:
            parser.error(f"not an IP address or email: {args.ip}")
        investigate(src, settings_, ip, hours, now)
        return 0

    send = telegram(settings_, args.dry_run)
    state = load_state(settings_)
    if args.command == "check":
        check(src, settings_, state, now, send)
        # Only the check saves state: a summary running at the same time
        # would otherwise overwrite what the check just recorded
        if not args.dry_run:
            save_state(settings_, state)
    else:
        if args.command in ("summary", "daily"):
            daily(src, settings_, state, now, send, persist=not args.dry_run)
        saturday = dt.datetime.fromtimestamp(now / 1e9, TZ).weekday() == 5
        if args.command == "weekly" or (args.command == "summary" and saturday):
            weekly(src, settings_, state, now, send, persist=not args.dry_run)
    return 0


if __name__ == "__main__":
    sys.exit(main())
