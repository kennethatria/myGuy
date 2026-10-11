import unittest

import myguy_bot as mb

ADMIN, OTHER, CHANNEL = 42, 7, -100123
CHAT = ADMIN  # a private chat's id is the other person's user id


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

        self.bot = mb.Bot({"admin": ADMIN}, self.tg, admin,
                          reports=lambda args: "report for " + " ".join(args),
                          now=lambda: self.clock)

    def message(self, text, user=ADMIN, chat=CHAT):
        self.bot.handle({"message": {"message_id": 5, "text": text, "chat": {"id": chat}, "from": {"id": user}}})

    def press(self, which="ok", user=ADMIN):
        data = self.tg.buttons()[-1]["inline_keyboard"][0][0 if which == "ok" else 1]["callback_data"]
        self.bot.handle({"callback_query": {"id": "q", "data": data, "from": {"id": user},
                                            "message": {"message_id": 99, "chat": {"id": CHAT}}}})

    def test_only_the_admin_in_their_private_chat(self):
        self.message("/ban 1.1.1.1", user=OTHER)              # someone else
        self.message("/ban 1.1.1.1", user=OTHER, chat=OTHER)  # in their own chat with the bot
        self.message("/ban 1.1.1.1", chat=CHANNEL)            # in a group or channel
        # A channel post has no sender at all
        self.bot.handle({"channel_post": {"message_id": 5, "text": "/ban 1.1.1.1", "chat": {"id": CHANNEL}}})
        self.message("hello")
        self.assertEqual(self.tg.calls, [])
        self.assertEqual(self.commands, [])

    def test_changes_ask_first_and_delete_the_command(self):
        self.message("/ban@MyGuyBot 1.1.1.1")
        self.assertEqual(self.tg.calls[0], ("deleteMessage", {"chat_id": CHAT, "message_id": 5}))
        self.assertEqual(self.commands, [], "nothing happens before Confirm")
        self.assertIn("Ban <code>1.1.1.1</code> for 7 days?", self.tg.sent()[-1])

        self.answer = {"ok": True, "ip": "1.1.1.1", "days": 7}
        self.press("ok")
        self.assertEqual(self.commands, ["ban 1.1.1.1"])
        self.assertIn("🚫 Banned <code>1.1.1.1</code> for 7 days.", self.tg.sent()[-1])

    def test_blocking_is_not_done_from_telegram(self):
        self.message("/block jane@example.com 30 spam")
        self.message("/unblock jane@example.com")
        self.assertEqual(self.commands, [])
        self.assertEqual(self.tg.buttons(), [])
        for text in self.tg.sent():
            self.assertIn("over SSH on the app server", text)
            self.assertNotIn("jane@example.com", text)

    def test_suspend_asks_then_says_how_long(self):
        self.message("/suspend Jane@Example.com spam again")
        question = self.tg.sent()[-1]
        self.assertIn("Suspend <code>j***@example.com</code> for the next length on the ladder (reason: spam again)?", question)
        self.assertEqual(self.commands, [])
        self.answer = {"ok": True, "until": "2026-10-12T00:00:00Z", "hours": 12, "step": 1}
        self.press("ok")
        self.assertEqual(self.commands, ["suspend jane@example.com spam again"])
        self.assertIn("⏸ Suspended <code>j***@example.com</code> until 2026-10-12 00:00 UTC (12 hours, 1st in a row)",
                      self.tg.sent()[-1])
        self.assertNotIn("consider a block", self.tg.sent()[-1])

    def test_suspend_with_a_length_and_the_fourth_suggests_a_block(self):
        self.message("/suspend jane@example.com 1w")
        self.assertIn("for 1 week?", self.tg.sent()[-1])
        self.answer = {"ok": True, "until": "2026-10-18T12:00:00Z", "hours": 168, "step": 4}
        self.press("ok")
        self.assertEqual(self.commands, ["suspend jane@example.com 1w"])
        self.assertIn("1 week, 4th in a row", self.tg.sent()[-1])
        self.assertIn("consider a block", self.tg.sent()[-1])

    def test_unsuspend(self):
        self.message("/unsuspend jane@example.com")
        self.assertIn("Lift the suspension of <code>j***@example.com</code>?", self.tg.sent()[-1])
        self.answer = {"ok": True}
        self.press("ok")
        self.assertEqual(self.commands, ["unsuspend jane@example.com"])
        self.assertIn("✅ Lifted the suspension", self.tg.sent()[-1])

    def test_status_shows_a_suspension(self):
        self.answer = {"ok": True, "blocked": False,
                       "suspension": {"suspended": True, "until": "2026-10-12T00:00:00Z", "reason": "spam", "in_row": 2}}
        self.message("/status jane@example.com")
        self.assertIn("isn't blocked.\nSuspended until 2026-10-12 00:00 UTC (spam), 2nd in a row.", self.tg.sent()[-1])
        self.answer = {"ok": True, "blocked": False, "suspension": {"suspended": False, "in_row": 1, "next_hours": 24}}
        self.message("/status jane@example.com")
        self.assertIn("Not suspended (1 in a row; the next would be 1 day).", self.tg.sent()[-1])

    def test_status_masks_the_email(self):
        self.answer = {"ok": True, "blocked": True, "until": None, "reason": "spam"}
        self.message("/status Jane@Example.com")
        self.assertEqual(self.commands, ["status jane@example.com"])
        self.assertIn("<code>j***@example.com</code> is blocked for good (spam).", self.tg.sent()[-1])

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
        self.message("/status not-an-email")
        self.assertEqual(self.commands, [])
        self.assertTrue(all(t.startswith("⚠️") for t in self.tg.sent()))

    def test_failure_is_shown(self):
        self.answer = {"ok": False, "error": "1.1.1.1 is not banned"}
        self.message("/unban 1.1.1.1")
        self.press("ok")
        self.assertIn("❌ 1.1.1.1 is not banned", self.tg.sent()[-1])

    def test_investigate(self):
        self.message("/investigate 1.1.1.1 48")
        self.assertIn("<pre>report for investigate 1.1.1.1 --hours 48</pre>", self.tg.sent()[-1])
        self.message("/investigate Jane@Example.com")
        self.assertIn("<pre>report for investigate jane@example.com</pre>", self.tg.sent()[-1])
        self.message("/investigate rm -rf")
        self.assertIn("⚠️", self.tg.sent()[-1])

    def test_caught(self):
        self.message("/caught")
        self.assertIn("<pre>report for caught</pre>", self.tg.sent()[-1])
        self.message("/caught 48")
        self.assertIn("<pre>report for caught --hours 48</pre>", self.tg.sent()[-1])
        self.message("/caught everything")
        self.assertIn("⚠️ usage: /caught [HOURS]", self.tg.sent()[-1])

    def test_health_and_now(self):
        self.message("/health")
        self.assertIn("<pre>report for health</pre>", self.tg.sent()[-1])
        self.message("/now")
        self.assertIn("<pre>report for now</pre>", self.tg.sent()[-1])
        self.message("/now please")
        self.assertIn("⚠️ usage: /now", self.tg.sent()[-1])

    def test_menu_is_for_the_admin_only(self):
        mb.set_menu(self.tg, ADMIN)
        method, params = self.tg.calls[-1]
        self.assertEqual(method, "setMyCommands")
        self.assertEqual(params["scope"], {"type": "chat", "chat_id": ADMIN})
        self.assertIn({"command": "health", "description": "Is everything OK right now"}, params["commands"])

    def test_untrusted_text_is_escaped(self):
        self.bot.reports = lambda args: "<script>"
        self.message("/investigate 1.1.1.1")
        self.assertIn("&lt;script&gt;", self.tg.sent()[-1])


class SettingsTest(unittest.TestCase):
    def test_key_comes_from_systemd_credentials(self):
        s = mb.settings({"TELEGRAM_CHAT_ID": "-100123", "TELEGRAM_ADMIN_USER_ID": "42",
                         "CREDENTIALS_DIRECTORY": "/run/credentials/myguy-bot.service"})
        self.assertEqual(s["admin"], 42)
        self.assertEqual(s["key"], "/run/credentials/myguy-bot.service/admin-key")

    def test_no_admin_obeys_nobody(self):
        bot = mb.Bot({"chat": CHAT, "admin": 0}, FakeTelegram(), lambda c: {"ok": True})
        self.assertFalse(bot.allowed(CHAT, 0))


if __name__ == "__main__":
    unittest.main()
