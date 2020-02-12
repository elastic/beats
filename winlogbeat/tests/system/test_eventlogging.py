import os
import sys
import time
import unittest
from winlogbeat import WriteReadTest

if sys.platform.startswith("win"):
    import win32security

"""
Contains tests for reading from the Event Logging API (pre MS Vista).
"""


@unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
class Test(WriteReadTest):

    @classmethod
    def setUpClass(self):
        self.api = "eventlogging"
        super(WriteReadTest, self).setUpClass()

    def test_read_one_event(self):
        """
        eventlogging - Read one classic event
        """
        msg = "Hello world!"
        self.write_event_log(msg)
        evts = self.read_events()
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg)

    def test_resume_reading_events(self):
        """
        eventlogging - Resume reading events
        """
        msg = "First event"
        self.write_event_log(msg)
        evts = self.read_events()
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg)

        # remove the output file, otherwise there is a race condition
        # in read_events() below where it reads the results of the previous
        # execution
        os.unlink(os.path.join(self.working_dir, "output", self.beat_name))

        msg = "Second event"
        self.write_event_log(msg)
        evts = self.read_events()
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg)

    def test_read_unknown_event_id(self):
        """
        eventlogging - Read unknown event ID
        """
        msg = "Unknown event ID"
        event_id = 1111
        self.write_event_log(msg, eventID=event_id)
        evts = self.read_events()
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], eventID=event_id)
        self.assertEqual(evts[0]["error.message"].lower(),
                         ("The system cannot find message text for message "
                          "number 1111 in the message file for "
                          "C:\\Windows\\system32\\EventCreate.exe.").lower())

    def test_read_unknown_sid(self):
        """
        eventlogging - Read event with unknown SID
        """
        # Fake SID that was made up.
        accountIdentifier = "S-1-5-21-3623811015-3361044348-30300820-1013"
        sid = win32security.ConvertStringSidToSid(accountIdentifier)

        msg = "Unknown SID " + accountIdentifier
        self.write_event_log(msg, sid=sid)
        evts = self.read_events()
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg, sid=accountIdentifier)

    def test_fields_under_root(self):
        """
        eventlogging - Add tags and custom fields under root
        """
        msg = "Add tags and fields under root"
        self.write_event_log(msg)
        evts = self.read_events(config={
            "tags": ["global"],
            "fields": {"global": "field", "env": "prod", "log.level": "overwrite"},
            "fields_under_root": True,
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                    "tags": ["local"],
                    "fields_under_root": True,
                    "fields": {"local": "field", "env": "dev"}
                }
            ]
        })
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg, level="overwrite", extra={
            "global": "field",
            "env": "dev",
            "local": "field",
            "tags": ["global", "local"],
        })

    def test_fields_not_under_root(self):
        """
        eventlogging - Add custom fields (not under root)
        """
        msg = "Add fields (not under root)"
        self.write_event_log(msg)
        evts = self.read_events(config={
            "fields": {"global": "field", "env": "prod", "level": "overwrite"},
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                    "fields": {"local": "field", "env": "dev", "num": 1}
                }
            ]
        })
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg, extra={
            "log.level": "information",
            "fields.global": "field",
            "fields.env": "dev",
            "fields.level": "overwrite",
            "fields.local": "field",
            "fields.num": 1,
        })
        self.assertTrue("tags" not in evts[0])

    def test_ignore_older(self):
        """
        eventlogging - Query by time (ignore_older than 4s)
        """
        self.write_event_log(">=4 seconds old", eventID=20)
        time.sleep(4)
        self.write_event_log("~0 seconds old", eventID=10)
        evts = self.read_events(config={
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                    "ignore_older": "2s"
                }
            ]
        }, expected_events=1)
        self.assertTrue(len(evts), 1)
        self.assertEqual(evts[0]["winlog.event_id"], 10)
        self.assertEqual(evts[0]["event.code"], 10)

    def test_unknown_eventlog_config(self):
        """
        eventlogging - Unknown config parameter
        """
        self.render_config_template(
            event_logs=[
                {
                    "name": self.providerName,
                    "api": self.api,
                    "event_id": "10, 12",
                    "level": "info",
                    "provider": ["me"],
                    "include_xml": True,
                }
            ]
        )
        self.start_beat().check_wait(exit_code=1)
        assert self.log_contains("4 errors: invalid event log key")

    def test_utf16_characters(self):
        """
        eventlogging - UTF-16 characters
        """
        msg = (u'\u89E3\u51CD\u3057\u305F\u30D5\u30A9\u30EB\u30C0\u306E'
               u'\u30A4\u30F3\u30B9\u30C8\u30FC\u30EB\u30B9\u30AF\u30EA'
               u'\u30D7\u30C8\u3092\u5B9F\u884C\u3057'
               u'\u8C61\u5F62\u5B57')
        self.write_event_log(msg)
        evts = self.read_events(config={
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                }
            ]
        })
        self.assertTrue(len(evts), 1)
        self.assertEqual(evts[0]["message"], msg)

    def test_registry_data(self):
        """
        eventlogging - Registry is updated
        """
        self.write_event_log("Hello world!")
        evts = self.read_events()
        self.assertTrue(len(evts), 1)

        event_logs = self.read_registry(requireBookmark=False)
        self.assertTrue(len(event_logs.keys()), 1)
        self.assertIn(self.providerName, event_logs)
        record_number = event_logs[self.providerName]["record_number"]
        self.assertGreater(record_number, 0)

    def test_processors(self):
        """
        eventlogging - Processors are applied
        """
        self.write_event_log("Hello world!")

        config = {
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                    "extras": {
                        "processors": [
                            {
                                "drop_fields": {
                                    "fields": ["message"],
                                }
                            }
                        ],
                    },
                }
            ]
        }
        evts = self.read_events(config)
        self.assertTrue(len(evts), 1)
        self.assertNotIn("message", evts[0])

    def test_multiline_events(self):
        """
        eventlogging - Event with newlines and control characters
        """
        msg = """
A trusted logon process has been registered with the Local Security Authority.
This logon process will be trusted to submit logon requests.

Subject:

Security ID:  SYSTEM
Account Name:  MS4\x1e$
Account Domain:  WORKGROUP
Logon ID:  0x3e7
Logon Process Name:  IKE"""
        self.write_event_log(msg)
        evts = self.read_events()
        self.assertTrue(len(evts), 1)
        self.assertEqual(unicode(self.api), evts[0]["winlog.api"], evts[0])
        self.assertNotIn("event.original", evts[0], msg=evts[0])
        self.assertIn("message", evts[0], msg=evts[0])
        self.assertNotIn("\\u000a", evts[0]["message"], msg=evts[0])
        self.assertEqual(unicode(msg), evts[0]["message"].decode('unicode-escape'), msg=evts[0])
