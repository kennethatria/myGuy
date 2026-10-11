import unittest

import myguy_admin as ma


class ParseTest(unittest.TestCase):
    def test_commands(self):
        self.assertEqual(ma.parse("ban 1.1.1.1"), ("ban", {"ip": "1.1.1.1"}))
        self.assertEqual(ma.parse("UNBAN 2606:4700::1111")[0], "unban")
        self.assertEqual(ma.parse("block Jane@Example.com"),
                         ("block", {"email": "jane@example.com", "days": 0, "reason": ""}))
        self.assertEqual(ma.parse("block jane@example.com 30 spam, fake listings"),
                         ("block", {"email": "jane@example.com", "days": 30, "reason": "spam, fake listings"}))
        self.assertEqual(ma.parse("blocked"), ("blocked", {}))

    def test_refuses_what_isnt_allowed(self):
        for command in [
            "", "rm -rf /", "ban", "ban 1.2.3.4 5.6.7.8", "ban not-an-ip",
            # ours or private: banning them would take the site down
            "ban 10.0.0.3", "ban 192.168.255.4", "ban 127.0.0.1", "ban 172.16.0.1",
            "block", "block nope", "block a@b.com 99999", "block a@b.com 7 $(reboot)", "block a@b.com 7 `id`",
            "block a@b.com 7 <b>x</b>", "unblock", "status a@b.com extra", "blocked please",
        ]:
            with self.assertRaises(ma.Refused, msg=command):
                ma.parse(command)

    def test_the_bot_cant_block(self):
        for command in ["block a@b.com", "unblock a@b.com", "BLOCK a@b.com 7 spam"]:
            with self.assertRaisesRegex(ma.Refused, "over SSH", msg=command):
                ma.parse(command, bot=True)
        self.assertEqual(ma.parse("status a@b.com", bot=True), ("status", {"email": "a@b.com"}))
        self.assertEqual(ma.parse("ban 1.1.1.1", bot=True), ("ban", {"ip": "1.1.1.1"}))

    def test_reason_length(self):
        with self.assertRaises(ma.Refused):
            ma.parse("block a@b.com 1 " + "x" * 201)


class RunTest(unittest.TestCase):
    def setUp(self):
        self.f2b_calls, self.http_calls = [], []

    def f2b(self, *args):
        self.f2b_calls.append(args)
        return "[['tripwire', 'manual']]" if args[0] == "banned" else ""

    def call(self, status=200, body=None):
        def fake(method, path, payload=None):
            self.http_calls.append((method, path, payload))
            return status, body or {}
        return fake

    def test_ip_actions(self):
        ma.run("ban", {"ip": "1.1.1.1"}, f2b=self.f2b)
        ma.run("unban", {"ip": "1.1.1.1"}, f2b=self.f2b)
        result = ma.run("ipstatus", {"ip": "1.1.1.1"}, f2b=self.f2b)
        self.assertEqual(self.f2b_calls, [("set", "manual", "banip", "1.1.1.1"),
                                          ("unban", "1.1.1.1"), ("banned", "1.1.1.1")])
        self.assertEqual(result["jails"], ["tripwire", "manual"])

    def test_email_actions_call_the_backend(self):
        ma.run("block", {"email": "a@b.com", "days": 0, "reason": "spam"}, call=self.call(body={"blocked": True}))
        ma.run("unblock", {"email": "a+x@b.com"}, call=self.call())
        ma.run("status", {"email": "a@b.com"}, call=self.call())
        ma.run("blocked", {}, call=self.call())
        self.assertEqual(self.http_calls, [
            ("POST", "/email-blocks", {"email": "a@b.com", "days": 0, "reason": "spam", "by": "telegram"}),
            ("DELETE", "/email-blocks/a%2Bx%40b.com", None),
            ("GET", "/email-blocks/a%40b.com", None),
            ("GET", "/email-blocks", None)])

    def test_backend_refusal_is_passed_on(self):
        result = ma.run("unblock", {"email": "a@b.com"}, call=self.call(404, {"error": "that address isn't blocked"}))
        self.assertEqual(result, {"ok": False, "error": "that address isn't blocked"})

    def test_main_prints_json(self):
        import io
        import contextlib
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            code = ma.main(["ban 10.0.0.2"])
        self.assertEqual(code, 1)
        self.assertFalse(__import__("json").loads(out.getvalue())["ok"])

    def test_main_with_bot_refuses_block(self):
        import io
        import contextlib
        from unittest import mock
        out = io.StringIO()
        with contextlib.redirect_stdout(out), mock.patch.object(ma, "run") as run:
            code = ma.main(["--bot", "block a@b.com"])
        self.assertEqual(code, 1)
        run.assert_not_called()
        self.assertIn("over SSH", __import__("json").loads(out.getvalue())["error"])

    def test_main_over_ssh_says_who(self):
        import io
        import contextlib
        from unittest import mock
        with contextlib.redirect_stdout(io.StringIO()), mock.patch.dict("os.environ", {"SUDO_USER": "myguy"}), \
                mock.patch.object(ma, "run", return_value={"ok": True}) as run, mock.patch.object(ma.syslog, "syslog"):
            ma.main(["block", "a@b.com", "7", "spam"])
        run.assert_called_once_with("block", {"email": "a@b.com", "days": 7, "reason": "spam"}, by="ssh:myguy")


class KeyTest(unittest.TestCase):
    def test_reads_the_internal_key(self):
        import tempfile
        with tempfile.NamedTemporaryFile("w", delete=False) as f:
            f.write("JWT_SECRET=x\nINTERNAL_API_KEY=abc=123\n")
        self.assertEqual(ma.internal_key(f.name), "abc=123")


if __name__ == "__main__":
    unittest.main()
