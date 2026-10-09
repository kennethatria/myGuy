"""Tests for myguy_security.py: python3 -m unittest discover -s configuration_management/files"""

import datetime as dt
import json
import os
import tempfile
import unittest

import myguy_security as ms

MIN, DAY = ms.MIN, ms.DAY
# Sat 17 Oct 2026, 08:00 in Amsterdam (06:00 UTC)
NOW = int(dt.datetime(2026, 10, 17, 6, 0, tzinfo=dt.timezone.utc).timestamp()) * 10**9


def waf_entry(ip, path="/api/v1/tasks?q=1", status=404, rules=("942100",), agent="sqlmap/1.7",
              flagged=True, entry="AAAA1111"):
    """A joined ModSecurity Serial audit entry, as Promtail ships it."""
    lines = [f"---{entry}---A--",
             f"[17/Oct/2026:06:00:00 +0000] 1791.1 {ip} 45616 192.168.128.141 443",
             f"---{entry}---B--", f"GET {path} HTTP/1.1", f"user-agent: {agent}", "host: akalimu.com", "",
             f"---{entry}---F--", f"HTTP/1.1 {status}", "Content-Type: text/html", "",
             f"---{entry}---H--"]
    for rule in rules:
        lines.append(f'ModSecurity: Warning. Matched "x" [file "y.conf"] [line "1"] [id "{rule}"] [msg "m"]')
    if flagged:
        lines.append('ModSecurity: Warning. Matched "x" [id "949110"] [msg "Inbound Anomaly Score Exceeded"]')
    lines += ["", f"---{entry}---Z--"]
    return "\n".join(lines)


class FakeSources:
    """Answers queries from canned data, keyed by job (and filters)."""

    def __init__(self):
        self.logs = {}       # job -> [(ts, labels, line)]
        self.vectors = []    # (substring of query, [(labels, value)])

    def add(self, job, ts, line, **labels):
        self.logs.setdefault(job, []).append((ts, labels, line))

    def lines(self, query, start, end):
        job = query.split('job="')[1].split('"')[0]
        out = [x for x in self.logs.get(job, []) if start <= x[0] < end]
        for flt in query.split("|=")[1:]:
            needle = flt.strip().split('"', 1)[1].rsplit('"', 1)[0].replace('\\"', '"')
            out = [x for x in out if needle in x[2]]
        return sorted(out, key=lambda x: x[0])

    def loki_vector(self, query, at):
        for needle, vector in self.vectors:
            if needle in query:
                return vector
        return []

    prom_vector = loki_vector


class TempState(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.TemporaryDirectory()
        self.settings = {"state_dir": self.dir.name, "app_ssh_host": "172.0.0.1", "app_vpc_ip": "10.0.0.2"}
        self.sent = []
        self.src = FakeSources()

    def tearDown(self):
        self.dir.cleanup()

    def send(self, text):
        self.sent.append(text)


class ParseTest(unittest.TestCase):
    def test_waf_entry(self):
        d = ms.parse_waf(1, waf_entry("203.0.113.7", status=200))
        self.assertEqual(d["ip"], "203.0.113.7")
        self.assertEqual(d["path"], "/api/v1/tasks", "query string dropped")
        self.assertEqual((d["method"], d["status"], d["agent"]), ("GET", 200, "sqlmap/1.7"))
        self.assertTrue(d["flagged"] and d["serious"])
        self.assertEqual(d["kinds"], ["SQL injection"])

    def test_low_score_protocol_warning_is_not_flagged(self):
        d = ms.parse_waf(1, waf_entry("203.0.113.7", rules=("920170",), flagged=False))
        self.assertFalse(d["flagged"] or d["serious"])
        self.assertEqual(d["kinds"], ["protocol"])

    def test_fail2ban(self):
        line = "2026-10-17 06:00:00,201 fail2ban.actions [812]: NOTICE  [nginx-modsecurity] Ban 203.0.113.7"
        self.assertEqual(ms.parse_fail2ban(1, line),
                         {"ts": 1, "jail": "nginx-modsecurity", "action": "Ban", "ip": "203.0.113.7"})
        self.assertIsNone(ms.parse_fail2ban(1, "NOTICE  [sshd] Ban not-an-ip"))

    def test_falco(self):
        line = json.dumps({"rule": "Terminal shell in container", "priority": "Notice",
                           "output_fields": {"proc.name": "bash", "container.name": "myguy_api_1"}})
        e = ms.parse_falco(1, line)
        self.assertEqual((e["rule"], e["process"], e["container"]), ("Terminal shell in container", "bash", "myguy_api_1"))
        self.assertIsNone(ms.parse_falco(1, "Falco initialized"))

    def test_untrusted_text_is_escaped_shortened_and_not_a_link(self):
        out = ms.code("<a href='https://evil.example'>click</a>" + "x" * 100, 30)
        self.assertTrue(out.startswith("<code>&lt;a href") and out.endswith("…</code>"))
        self.assertNotIn("<a ", out)


class AttackTest(TempState):
    def hits(self, ip, n, start, status=404, rules=("942100",)):
        for i in range(n):
            self.src.add("modsecurity", start + i * MIN // 2, waf_entry(ip, status=status, rules=rules))

    def check(self, now=NOW, state=None):
        state = {} if state is None else state
        ms.check(self.src, self.settings, state, now, self.send)
        return state

    def test_banned_attacker_is_left_to_fail2ban(self):
        self.hits("203.0.113.7", 8, NOW - 9 * MIN)
        self.src.add("fail2ban", NOW - 8 * MIN, "NOTICE  [nginx-modsecurity] Ban 203.0.113.7", jail="nginx-modsecurity")
        self.check()
        self.assertEqual(self.sent, [])

    def test_unbanned_attacker_needs_a_look_once(self):
        self.hits("203.0.113.7", 8, NOW - 9 * MIN)
        state = self.check()
        self.assertEqual(len(self.sent), 1)
        msg = self.sent[0]
        self.assertIn("<code>203.0.113.7</code>", msg)
        self.assertIn("fail2ban hasn't banned it", msg)
        self.assertIn("SQL injection", msg)
        self.assertIn("sudo fail2ban-client set manual banip 203.0.113.7", msg)
        self.check(NOW + 5 * MIN, state)
        self.assertEqual(len(self.sent), 1, "nothing new: no repeat")

    def test_new_attacker_gets_a_grace_period_for_fail2ban(self):
        self.hits("203.0.113.7", 8, NOW - 3 * MIN)
        self.check()
        self.assertEqual(self.sent, [])

    def test_succeeded_serious_request_is_sent_at_once_even_if_banned(self):
        self.src.add("modsecurity", NOW - MIN, waf_entry("203.0.113.7", status=200, agent="<script>evil.example</script>"))
        self.src.add("fail2ban", NOW - MIN, "NOTICE  [manual] Ban 203.0.113.7", jail="manual")
        self.check()
        self.assertEqual(len(self.sent), 1)
        self.assertIn("answered without an error", self.sent[0])
        self.assertIn("<code>&lt;script&gt;evil.example&lt;/script&gt;</code>", self.sent[0])
        self.assertNotIn("<script>", self.sent[0])

    def test_sign_in_abuse(self):
        self.src.vectors.append(("/api/v1/auth/", [({"remote_addr": "198.51.100.2"}, 12)]))
        self.check()
        self.assertIn("sign-in calls", self.sent[0])
        self.assertIn("<code>198.51.100.2</code>", self.sent[0])

    def test_attack_ending_is_summarised(self):
        self.hits("203.0.113.7", 8, NOW - 9 * MIN)
        state = self.check()
        later = NOW + 45 * MIN
        self.check(later, state)
        self.assertEqual(len(self.sent), 2)
        self.assertIn("Attack ended", self.sent[1])
        self.assertIn("8 flagged in total", self.sent[1])
        self.assertNotIn("203.0.113.7", state["attacks"])

    def test_heartbeat(self):
        import contextlib
        import io
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            self.check()
        self.assertIn("check ok", out.getvalue())


class SshTest(TempState):
    def login(self, ts, user, ip, host="app"):
        self.src.add("ssh", ts, f"Accepted publickey for {user} from {ip} port 50000 ssh2: ED25519 SHA256:abc", host=host)

    def check(self, now=NOW, state=None):
        state = {"ssh_since": NOW - 10 * MIN} if state is None else state
        ms.check(self.src, self.settings, state, now, self.send)
        return state

    def test_ci_key_outside_a_deploy(self):
        self.login(NOW - 2 * MIN, "myguy", "20.1.2.3")
        self.check()
        self.assertIn("no deploy was running", self.sent[0])

    def test_ci_key_during_a_deploy(self):
        self.src.add("deploy", NOW - 8 * MIN, "start run=42 scope=app services=api")
        self.login(NOW - 7 * MIN, "myguy", "20.1.2.3")
        self.login(NOW - 6 * MIN, "myguy", "10.0.0.2", host="monitoring")
        self.check()
        self.assertEqual(self.sent, [])

    def test_ops_from_a_new_ip_once(self):
        self.login(NOW - 2 * MIN, "ops", "198.51.100.9")
        state = self.check()
        self.assertIn("not seen in the last 8 days", self.sent[0])
        self.login(NOW + 2 * MIN, "ops", "198.51.100.9")
        self.check(NOW + 5 * MIN, state)
        self.assertEqual(len(self.sent), 1, "known now")

    def test_monitoring_only_through_the_app_server(self):
        self.login(NOW - 2 * MIN, "ops", "10.0.0.2", host="monitoring")
        self.login(NOW - 1 * MIN, "ops", "198.51.100.9", host="monitoring")
        self.check()
        self.assertEqual(len(self.sent), 1)
        self.assertIn("<code>198.51.100.9</code>", self.sent[0])
        self.assertNotIn("10.0.0.2</code>", self.sent[0])

    def test_lines_seen_twice_alert_once(self):
        self.login(NOW - 1 * MIN, "root", "198.51.100.9")
        state = self.check()
        state["ssh_since"] = NOW - 5 * MIN   # overlapping window
        self.check(NOW + 1, state)
        self.assertEqual(len(self.sent), 1)


class SummaryTest(TempState):
    def populate(self):
        s = self.src
        s.vectors += [("avg_over_time(probe_success", [({}, 1.0)]),
                      ("sum_over_time((1 - min", [({}, 0)]),
                      ("probe_ssl_earliest", [({}, 61)]),
                      ("node_filesystem", [({"instance": "app"}, 42), ({"instance": "monitoring"}, 38)]),
                      ("node_memory", [({"instance": "app"}, 81), ({"instance": "monitoring"}, 77)]),
                      ("topk(1, sum by (method, uri)", [({"method": "GET", "uri": "/items"}, 3)]),
                      ("status >= 500", [({}, 3)]),
                      ("request-code", [({}, 38)]),
                      ("status=429", [({}, 2)]),
                      ("count by (remote_addr) (count_over_time({job=\"nginx_access\"} | json | uri =~",
                       [({"remote_addr": f"192.0.2.{i}"}, 1) for i in range(1, 5)]),
                      ("myguy_accounts", [({}, 1)])]
        for i in range(5):
            s.add("modsecurity", NOW - 3 * 60 * MIN + i * MIN, waf_entry("203.0.113.7", entry=f"A{i}"))
        s.add("modsecurity", NOW - 60 * MIN, waf_entry("198.51.100.2", rules=("913100",), entry="B1"))
        s.add("fail2ban", NOW - 170 * MIN, "NOTICE  [nginx-modsecurity] Ban 203.0.113.7", jail="nginx-modsecurity")
        s.add("ssh", NOW - 100 * MIN, "Accepted publickey for ops from 198.51.100.9 port 1 ssh2", host="app")
        s.add("ssh", NOW - 90 * MIN, "Invalid user admin from 192.0.2.44 port 2", host="app")
        s.add("sudo", NOW - 99 * MIN, "     ops : PWD=/tmp ; USER=root ; COMMAND=/usr/local/bin/appctl ps", host="app")
        s.add("sudo", NOW - 98 * MIN, "   myguy : PWD=/opt ; USER=root ; COMMAND=/bin/sh -c echo", host="app")
        s.add("falco", NOW - 50 * MIN, json.dumps({"rule": "Read sensitive file untrusted", "priority": "Warning"}))
        s.add("deploy", NOW - 300 * MIN, "start run=42 scope=app services=api,store-service")

    def test_daily_compares_with_yesterday_and_saves(self):
        self.populate()
        yesterday = {"kind": "daily", "date": "2026-10-16", "stats": {"waf": 9, "visitors": 4, "server_errors": 5}}
        ms.save_history(self.settings, [yesterday], dt.date(2026, 10, 17))
        ms.daily(self.src, self.settings, {}, NOW, self.send)
        msg = self.sent[0]
        self.assertIn("MyGuy daily summary: Sat 17 Oct", msg)
        self.assertIn("Site up 100% · certificate valid 61 days", msg)
        self.assertIn("WAF detections: 6 ↓ from 9 from 2 IPs", msg)
        self.assertIn("~4 real visitors =", msg)
        self.assertIn("Server errors: 3 ↓ from 5", msg)
        self.assertIn("<code>203.0.113.7</code> 5 hits · SQL injection · banned", msg)
        self.assertIn("ops@app ×1", msg)
        self.assertIn("Failed SSH attempts: 1", msg)
        self.assertIn("sudo by ops: 1", msg)
        self.assertIn("Deploys: 1", msg)
        self.assertIn("Needs attention:</b> nothing ✅", msg)
        saved = ms.load_history(self.settings)
        self.assertEqual([r["date"] for r in saved], ["2026-10-16", "2026-10-17"])
        self.assertEqual(saved[-1]["stats"]["banned_ips"], ["203.0.113.7"])

    def test_needs_attention(self):
        self.populate()
        self.src.add("modsecurity", NOW - MIN, waf_entry("203.0.113.9", status=200, entry="C1"))
        ms.daily(self.src, self.settings, {"ssh_alert_log": [NOW - MIN]}, NOW, self.send)
        self.assertIn("1 unexpected SSH login(s)", self.sent[0])
        self.assertIn("1 serious attack request(s) answered without an error", self.sent[0])

    def test_unreadable_source_is_named(self):
        class Broken(FakeSources):
            def prom_vector(self, query, at):
                raise OSError("down")
        self.src = Broken()
        ms.daily(self.src, self.settings, {}, NOW, self.send)
        self.assertIn("couldn't read: health", self.sent[0])

    def test_weekly_adds_up_the_days(self):
        records = []
        for n in range(7):
            date = dt.date(2026, 10, 17) - dt.timedelta(days=n)
            records.append({"kind": "daily", "date": date.isoformat(), "stats": {
                "waf": 10 + n, "uptime": 100.0, "bans": {"nginx-modsecurity": 1},
                "banned_ips": ["203.0.113.7"] if n < 2 else [], "kinds": {"scanner": 3, "SQL injection": 1},
                "deploys": [["13:20", "app"]], "top_ips": [["203.0.113.7", 5, "scanner", True]]}})
        records.append({"kind": "weekly", "date": "2026-10-10", "stats": {"waf": 50}})
        ms.save_history(self.settings, records, dt.date(2026, 10, 17))
        self.src.vectors.append(("[7d]", [({"remote_addr": f"192.0.2.{i}"}, 1) for i in range(22)]))
        ms.weekly(self.src, self.settings, {}, NOW, self.send)
        msg = self.sent[0]
        self.assertIn("weekly summary: Sun 11 Oct – Sat 17 Oct", msg)
        self.assertIn("WAF detections: 91 ↑ from 50", msg)
        self.assertIn("Bans: 7", msg)
        self.assertIn("Repeat offenders: 1 IPs", msg)
        self.assertIn("Busiest day: Sun (16 detections)", msg)
        self.assertIn("scanner 75%", msg)
        self.assertIn("Deploys: 7", msg)
        self.assertIn("~22 real visitors", msg)

    def test_summary_sends_weekly_on_saturdays(self):
        self.populate()
        self.assertEqual(dt.datetime.fromtimestamp(NOW / 1e9, ms.TZ).weekday(), 5)
        ms.daily(self.src, self.settings, {}, NOW, self.send)
        ms.weekly(self.src, self.settings, {}, NOW, self.send)
        self.assertIn("6 day(s) with no daily summary", self.sent[1])


class InvestigateTest(TempState):
    def test_report(self):
        ip = "203.0.113.7"
        self.src.add("nginx_access", NOW - 5 * MIN, json.dumps({
            "remote_addr": ip, "method": "POST", "uri": "/api/v1/auth/request-code", "status": 429, "user_agent": "curl"},
            separators=(",", ":")))  # as nginx writes it
        self.src.add("modsecurity", NOW - 4 * MIN, waf_entry(ip, status=200))
        self.src.add("fail2ban", NOW - 3 * MIN, f"NOTICE  [nginx-modsecurity] Ban {ip}", jail="nginx-modsecurity")
        out = []
        ms.investigate(self.src, self.settings, ip, 24, NOW, out.append)
        text = "\n".join(out)
        self.assertIn("Requests: 1 (4xx 1)", text)
        self.assertIn("rate-limited (429): 1", text)
        self.assertIn("WAF: 1 flagged (SQL injection 1) · serious answered without an error: 1", text)
        self.assertIn("banned now by nginx-modsecurity", text)
        self.assertIn(f"banip {ip}", text)


COUNTRIES = {"203.0.113.7": "CN", "198.51.100.2": "US", "198.51.100.9": "NL", "192.0.2.44": "RU"}


class CountryTest(TempState):
    def setUp(self):
        super().setUp()
        self.settings["country"] = lambda ip: COUNTRIES.get(ip, "NL" if ip.startswith("192.0.2.") else "")

    def test_attack_message_names_the_country(self):
        for i in range(8):
            self.src.add("modsecurity", NOW - 9 * MIN + i * MIN // 2, waf_entry("203.0.113.7"))
        ms.check(self.src, self.settings, {}, NOW, self.send)
        self.assertIn("<code>203.0.113.7</code> (CN) · 8 flagged", self.sent[0])

    def test_ssh_alert_names_the_country(self):
        self.src.add("ssh", NOW - MIN, "Accepted publickey for ops from 198.51.100.9 port 1 ssh2", host="app")
        ms.check(self.src, self.settings, {"ssh_since": NOW - 10 * MIN}, NOW, self.send)
        self.assertIn("from <code>198.51.100.9</code> (NL)", self.sent[0])

    def test_summaries_count_countries_by_ip(self):
        SummaryTest.populate(self)
        ms.daily(self.src, self.settings, {}, NOW, self.send)
        msg = self.sent[0]
        self.assertIn("<code>203.0.113.7</code> (CN) 5 hits", msg)
        self.assertIn("Attacker countries (IPs): CN 1, RU 1, US 1", msg)
        self.assertIn("From: NL 4", msg)
        ms.weekly(self.src, self.settings, {}, NOW, self.send)
        self.assertIn("Attacker countries (IPs): CN 1, RU 1, US 1", self.sent[1])

    def test_investigate_names_the_country(self):
        out = []
        ms.investigate(self.src, self.settings, "203.0.113.7", 24, NOW, out.append)
        self.assertTrue(out[0].startswith("Investigate 203.0.113.7 (CN)"))

    def test_no_database_means_no_country(self):
        self.assertEqual(ms.country_lookup("/nonexistent/country.mmdb")("8.8.8.8"), "")
        self.assertEqual(ms.place({}, "8.8.8.8"), "<code>8.8.8.8</code>")

    @unittest.skipUnless(os.environ.get("GEOIP_TEST_DB"), "set GEOIP_TEST_DB to a DB-IP country .mmdb")
    def test_real_database(self):
        country = ms.country_lookup(os.environ["GEOIP_TEST_DB"])
        self.assertEqual((country("8.8.8.8"), country("10.0.0.5"), country("not-an-ip")), ("US", "", ""))


class SettingsTest(unittest.TestCase):
    def test_environment_wins_over_the_file(self):
        os.environ["LOKI_URL"] = "http://loki.test:3100"
        try:
            self.assertEqual(ms.settings()["loki"], "http://loki.test:3100")
        finally:
            del os.environ["LOKI_URL"]


if __name__ == "__main__":
    unittest.main()
