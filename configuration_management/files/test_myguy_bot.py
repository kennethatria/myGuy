import unittest

import myguy_bot as mb

CHAT, ADMIN, OTHER = -100123, 42, 7


class FakeTelegram:
    def __init__(self):
        self.calls = []

    def call(self, method, **params):
        self.calls.append((method, params))
        return {"message_id": 99}

    def sent(self):
        return [p["text"] for m, p in self.calls if m in ("sendMessage", "editMessageText")]

    def buttons(self):
        return [p["reply_markup"] for m, p in self.calls if m == "sendMessage" and "reply_markup" in p]


class BotTest(unittest.TestCase):
    def setUp(self):
        self.tg = FakeTelegram()
        self.commands = []
        self.answer = {"ok": True}
        self.clock = 1000.0

        def admin(command):
            self.commands.append(command)
            return self.answer

        self.bot = mb.Bot({"chat": CHAT, "admin": ADMIN}, self.tg, admin,
                          investigator=lambda target, hours: f"report for {target} {hours}",
                          now=lambda: self.clock)

    def message(self, text, user=ADMIN, chat=CHAT):
        self.bot.handle({"message": {"message_id": 5, "text": text, "chat": {"id": chat}, "from": {"id": user}}})

    def press(self, which="ok", user=ADMIN):
        data = self.tg.buttons()[-1]["inline_keyboard"][0][0 if which == "ok" else 1]["callback_data"]
        self.bot.handle({"callback_query": {"id": "q", "data": data, "from": {"id": user},
                                            "message": {"message_id": 99, "chat": {"id": CHAT}}}})

    def test_only_the_admin_in_the_group(self):
        self.message("/ban 1.1.1.1", user=OTHER)
        self.message("/ban 1.1.1.1", chat=12345)
        self.message("hello")
        self.assertEqual(self.tg.calls, [])
        self.assertEqual(self.commands, [])

    def test_changes_ask_first_and_delete_the_command(self):
        self.message("/block@MyGuyBot Jane@Example.com 30 spam")
        self.assertEqual(self.tg.calls[0], ("deleteMessage", {"chat_id": CHAT, "message_id": 5}))
        self.assertEqual(self.commands, [], "nothing happens before Confirm")
        question = self.tg.sent()[-1]
        self.assertIn("Block <code>j***@example.com</code> for 30 days (reason: spam)?", question)
        self.assertNotIn("jane@example.com", question)

        self.answer = {"ok": True, "until": "2026-11-10T12:00:00Z"}
        self.press("ok")
        self.assertEqual(self.commands, ["block jane@example.com 30 spam"])
        self.assertIn("⛔ Blocked <code>j***@example.com</code> until 2026-11-10", self.tg.sent()[-1])

    def test_cancel(self):
        self.message("/ban 1.1.1.1")
        self.press("no")
        self.assertEqual(self.commands, [])
        self.assertIn("Cancelled", self.tg.sent()[-1])

    def test_confirm_expires(self):
        self.message("/unban 1.1.1.1")
        self.clock += mb.CONFIRM_FOR + 1
        self.press("ok")
        self.assertEqual(self.commands, [])
        self.assertIn("Expired", self.tg.sent()[-1])

    def test_a_button_works_once(self):
        self.message("/ban 1.1.1.1")
        self.press("ok")
        self.press("ok")
        self.assertEqual(self.commands, ["ban 1.1.1.1"])

    def test_someone_else_cant_press_confirm(self):
        self.message("/ban 1.1.1.1")
        self.press("ok", user=OTHER)
        self.assertEqual(self.commands, [])
        self.press("ok")
        self.assertEqual(self.commands, ["ban 1.1.1.1"])

    def test_lookups_answer_at_once(self):
        self.answer = {"ok": True, "jails": ["tripwire"]}
        self.message("/ipstatus 1.1.1.1")
        self.assertEqual(self.commands, ["ipstatus 1.1.1.1"])
        self.assertIn("banned in: tripwire", self.tg.sent()[-1])

        self.answer = {"ok": True, "blocks": [{"email": "jane@example.com", "until": None, "reason": "spam"}]}
        self.message("/blocked")
        self.assertIn("• <code>j***@example.com</code> for good (spam)", self.tg.sent()[-1])

    def test_bad_input_is_refused_before_ssh(self):
        self.message("/ban 10.0.0.2")
        self.message("/block not-an-email")
        self.assertEqual(self.commands, [])
        self.assertTrue(all(t.startswith("⚠️") for t in self.tg.sent()))

    def test_failure_is_shown(self):
        self.answer = {"ok": False, "error": "that address isn't blocked"}
        self.message("/unblock jane@example.com")
        self.press("ok")
        self.assertIn("❌ that address isn&#x27;t blocked", self.tg.sent()[-1].replace("'", "&#x27;"))

    def test_investigate(self):
        self.message("/investigate 1.1.1.1 48")
        self.assertIn("<pre>report for 1.1.1.1 48</pre>", self.tg.sent()[-1])
        self.message("/investigate Jane@Example.com")
        self.assertIn("report for jane@example.com None", self.tg.sent()[-1])
        self.message("/investigate rm -rf")
        self.assertIn("⚠️", self.tg.sent()[-1])

    def test_untrusted_text_is_escaped(self):
        self.bot.investigate = lambda target, hours: "<script>"
        self.message("/investigate 1.1.1.1")
        self.assertIn("&lt;script&gt;", self.tg.sent()[-1])


class SettingsTest(unittest.TestCase):
    def test_key_comes_from_systemd_credentials(self):
        s = mb.settings({"TELEGRAM_CHAT_ID": "-100123", "TELEGRAM_ADMIN_USER_ID": "42",
                         "CREDENTIALS_DIRECTORY": "/run/credentials/myguy-bot.service"})
        self.assertEqual((s["chat"], s["admin"]), (-100123, 42))
        self.assertEqual(s["key"], "/run/credentials/myguy-bot.service/admin-key")

    def test_no_admin_obeys_nobody(self):
        bot = mb.Bot({"chat": CHAT, "admin": 0}, FakeTelegram(), lambda c: {"ok": True})
        self.assertFalse(bot.allowed(CHAT, 0))


if __name__ == "__main__":
    unittest.main()
