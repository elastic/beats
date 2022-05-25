import codecs
import os
import platform
import sys
import time
import unittest
from winlogbeat import WriteReadTest

if sys.platform.startswith("win"):
    import win32evtlog
    import win32security

"""
Contains tests for reading from the Windows Event Log API (MS Vista and newer).
"""


@unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
class Test(WriteReadTest):

    @classmethod
    def setUpClass(self):
        self.api = "wineventlog"
        super(WriteReadTest, self).setUpClass()

    def test_read_one_event(self):
        """
        wineventlog - Read one classic event
        """
        msg = "Hello world!"
        self.write_event_log(msg)
        evts = self.read_events()
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg, extra={
            "winlog.keywords": ["Classic"],
            "winlog.opcode": "Info",
        })

    def test_resume_reading_events(self):
        """
        wineventlog - Resume reading events
        """
        msg = "First event"
        self.write_event_log(msg)
        evts = self.read_events()
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg, extra={
            "winlog.keywords": ["Classic"],
            "winlog.opcode": "Info",
        })

        # remove the output file, otherwise there is a race condition
        # in read_events() below where it reads the results of the previous
        # execution
        os.unlink(os.path.join(self.working_dir, "output", self.beat_name + "-" + self.today + ".ndjson"))

        msg = "Second event"
        self.write_event_log(msg)
        evts = self.read_events()
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg, extra={
            "winlog.keywords": ["Classic"],
            "winlog.opcode": "Info",
        })

    def test_cleared_channel_restarts(self):
        """
        wineventlog - When a bookmark points to a cleared (stale) channel
        the subscription starts from the beginning
        """
        msg1 = "First event"
        self.write_event_log(msg1)
        msg2 = "Second event"
        self.write_event_log(msg2)

        evts = self.read_events(expected_events=2)

        self.assertTrue(len(evts), 2)
        self.assert_common_fields(evts[0], msg=msg1)
        self.assert_common_fields(evts[1], msg=msg2)

        # remove the output file, otherwise there is a race condition
        # in read_events() below where it reads the results of the previous
        # execution
        os.unlink(os.path.join(self.working_dir, "output", self.beat_name + "-" + self.today + ".ndjson"))

        self.clear_event_log()

        # we check that after clearing the event log the bookmark still points to the previous checkpoint
        event_logs = self.read_registry(requireBookmark=True)
        self.assertTrue(len(list(event_logs.keys())), 1)
        self.assertIn(self.providerName, event_logs)
        record_number = event_logs[self.providerName]["record_number"]
        self.assertTrue(record_number, 2)

        msg3 = "Third event"
        self.write_event_log(msg3)

        evts = self.read_events()
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg3)

    def test_bad_bookmark_restart(self):
        """
        wineventlog - When a bookmarked event does not exist the subcription
        restarts from the beginning
        """
        msg1 = "First event"
        self.write_event_log(msg1)

        evts = self.read_events(expected_events=1)

        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg1)

        event_logs = self.read_registry(requireBookmark=True)
        self.assertTrue(len(list(event_logs.keys())), 1)
        self.assertIn(self.providerName, event_logs)
        record_number = event_logs[self.providerName]["record_number"]
        self.assertTrue(record_number, 1)

        # write invalid bookmark, it should start from the beginning again
        f = open(os.path.join(self.working_dir, "data", ".winlogbeat.yml"), "w")
        f.write((
            "update_time: 2100-01-01T00:00:00Z\n" +
            "event_logs:\n" +
            "  - name: {}\n" +
            "    record_number: 1000\n" +
            "    timestamp: 2100-01-01T00:00:00Z\n" +
            "    bookmark: \"<BookmarkList>\\r\\n  <Bookmark Channel='{}' RecordId='1000' IsCurrent='true'/>\\r\\n</BookmarkList>\"\n").
            format(self.providerName, self.providerName)
        )
        f.close()

        # remove the output file, otherwise there is a race condition
        # in read_events() below where it reads the results of the previous
        # execution
        os.unlink(os.path.join(self.working_dir, "output", self.beat_name + "-" + self.today + ".ndjson"))

        evts = self.read_events(expected_events=1)
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg1)

    def test_read_unknown_event_id(self):
        """
        wineventlog - Read unknown event ID
        """
        msg = "Unknown event ID"
        self.write_event_log(msg, eventID=1111)
        evts = self.read_events()
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], eventID="1111", extra={
            "winlog.keywords": ["Classic"],
            "winlog.opcode": "Info",
        })
        # Oddly, no rendering error is being given.
        self.assertTrue("error.message" not in evts[0])

    def test_read_unknown_sid(self):
        """
        wineventlog - Read event with unknown SID
        """
        # Fake SID that was made up.
        accountIdentifier = "S-1-5-21-3623811015-3361044348-30300820-1013"
        sid = win32security.ConvertStringSidToSid(accountIdentifier)

        msg = "Unknown SID " + accountIdentifier
        self.write_event_log(msg, sid=sid)
        evts = self.read_events()
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg, sid=accountIdentifier, extra={
            "winlog.keywords": ["Classic"],
            "winlog.opcode": "Info",
        })

    def test_fields_under_root(self):
        """
        wineventlog - Add tags and custom fields under root
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
            "winlog.keywords": ["Classic"],
            "winlog.opcode": "Info",
            "global": "field",
            "env": "dev",
            "local": "field",
            "tags": ["global", "local"],
        })

    def test_fields_not_under_root(self):
        """
        wineventlog - Add custom fields (not under root)
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
            "winlog.keywords": ["Classic"],
            "winlog.opcode": "Info",
            "fields.global": "field",
            "fields.env": "dev",
            "fields.level": "overwrite",
            "fields.local": "field",
            "fields.num": 1,
        })
        self.assertTrue("tags" not in evts[0])

    def test_include_xml(self):
        """
        wineventlog - Include raw XML event
        """
        msg = "Include raw XML event"
        self.write_event_log(msg)
        evts = self.read_events(config={
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                    "include_xml": True,
                }
            ]
        })
        self.assertTrue(len(evts), 1)
        self.assert_common_fields(evts[0], msg=msg)
        self.assertTrue("event.original" in evts[0])
        original = evts[0]["event.original"]
        self.assertTrue(original.endswith('</Event>'),
                        'xml value should end with </Event>: "{}"'.format(original))

    def test_query_event_id(self):
        """
        wineventlog - Query by event IDs
        """
        msg = "event_id test case"
        self.write_event_log(msg, eventID=10)  # Excluded
        self.write_event_log(msg, eventID=50)
        self.write_event_log(msg, eventID=100)
        self.write_event_log(msg, eventID=150)  # Excluded
        self.write_event_log(msg, eventID=175)
        self.write_event_log(msg, eventID=200)
        evts = self.read_events(config={
            "tags": ["event_id"],
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                    "event_id": "50, 100-200, -150"
                }
            ]
        }, expected_events=4)
        self.assertTrue(len(evts), 4)
        self.assertEqual(evts[0]["winlog.event_id"], "50")
        self.assertEqual(evts[1]["winlog.event_id"], "100")
        self.assertEqual(evts[2]["winlog.event_id"], "175")
        self.assertEqual(evts[3]["winlog.event_id"], "200")

    def test_query_level_single(self):
        """
        wineventlog - Query by level (warning)
        """
        self.write_event_log("success", level=win32evtlog.EVENTLOG_SUCCESS)
        self.write_event_log("error", level=win32evtlog.EVENTLOG_ERROR_TYPE)
        self.write_event_log(
            "warning", level=win32evtlog.EVENTLOG_WARNING_TYPE)
        self.write_event_log(
            "information", level=win32evtlog.EVENTLOG_INFORMATION_TYPE)
        evts = self.read_events(config={
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                    "level": "warning"
                }
            ]
        })
        self.assertTrue(len(evts), 1)
        self.assertEqual(evts[0]["log.level"], "warning")

    def test_query_level_multiple(self):
        """
        wineventlog - Query by level (error, warning)
        """
        self.write_event_log(
            "success", level=win32evtlog.EVENTLOG_SUCCESS)  # Level 0, Info
        self.write_event_log(
            "error", level=win32evtlog.EVENTLOG_ERROR_TYPE)  # Level 2
        self.write_event_log(
            "warning", level=win32evtlog.EVENTLOG_WARNING_TYPE)  # Level 3
        self.write_event_log(
            "information", level=win32evtlog.EVENTLOG_INFORMATION_TYPE)  # Level 4
        evts = self.read_events(config={
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                    "level": "error, warning"
                }
            ]
        }, expected_events=2)
        self.assertTrue(len(evts), 2)
        self.assertEqual(evts[0]["log.level"], "error")
        self.assertEqual(evts[1]["log.level"], "warning")

    @unittest.skipIf(platform.platform().startswith("Windows-7"),
                     "Flaky test: https://github.com/elastic/beats/issues/22753")
    def test_query_ignore_older(self):
        """
        wineventlog - Query by time (ignore_older than 2s)
        """
        self.write_event_log(">=2 seconds old", eventID=20)
        time.sleep(2)
        self.write_event_log("~0 seconds old", eventID=10)
        evts = self.read_events(config={
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                    "ignore_older": "2s"
                }
            ]
        })
        self.assertTrue(len(evts), 1)
        self.assertEqual(evts[0]["winlog.event_id"], "10")
        self.assertEqual(evts[0]["event.code"], "10")

    def test_query_provider(self):
        """
        wineventlog - Query by provider name (event source)
        """
        self.write_event_log("selected", source=self.otherAppName)
        self.write_event_log("filtered")
        evts = self.read_events(config={
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                    "provider": [self.otherAppName]
                }
            ]
        })
        self.assertTrue(len(evts), 1)
        self.assertEqual(evts[0]["winlog.provider_name"], self.otherAppName)

    def test_query_multi_param(self):
        """
        wineventlog - Query by multiple params
        """
        self.write_event_log("selected", source=self.otherAppName,
                             eventID=556, level=win32evtlog.EVENTLOG_ERROR_TYPE)
        self.write_event_log("filtered", source=self.otherAppName, eventID=556)
        self.write_event_log(
            "filtered", level=win32evtlog.EVENTLOG_WARNING_TYPE)
        evts = self.read_events(config={
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                    "event_id": "10-20, 30-40, -35, -18, 400-1000, -432",
                    "level": "warn, error",
                    "provider": [self.otherAppName]
                }
            ]
        })
        self.assertTrue(len(evts), 1)
        self.assertEqual(evts[0]["message"], "selected")

    def test_utf16_characters(self):
        """
        wineventlog - UTF-16 characters
        """
        msg = (u'\u89E3\u51CD\u3057\u305F\u30D5\u30A9\u30EB\u30C0\u306E'
               u'\u30A4\u30F3\u30B9\u30C8\u30FC\u30EB\u30B9\u30AF\u30EA'
               u'\u30D7\u30C8\u3092\u5B9F\u884C\u3057'
               u'\u8C61\u5F62\u5B57')
        self.write_event_log(str(msg))
        evts = self.read_events(config={
            "event_logs": [
                {
                    "name": self.providerName,
                    "api": self.api,
                    "include_xml": True,
                }
            ]
        })
        self.assertTrue(len(evts), 1)
        self.assertEqual(evts[0]["message"], msg)

    def test_registry_data(self):
        """
        wineventlog - Registry is updated
        """
        self.write_event_log("Hello world!")
        evts = self.read_events()
        self.assertTrue(len(evts), 1)

        event_logs = self.read_registry(requireBookmark=True)
        self.assertTrue(len(list(event_logs.keys())), 1)
        self.assertIn(self.providerName, event_logs)
        record_number = event_logs[self.providerName]["record_number"]
        self.assertGreater(record_number, 0)

    def test_processors(self):
        """
        wineventlog - Processors are applied
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
        wineventlog - Event with newlines and control characters
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
        self.assertEqual(str(self.api), evts[0]["winlog.api"], msg=evts[0])
        self.assertNotIn("event.original", evts[0], msg=evts[0])
        self.assertIn("message", evts[0], msg=evts[0])
        self.assertNotIn("\\u000a", evts[0]["message"], msg=evts[0])
        self.assertEqual(str(msg), codecs.decode(evts[0]["message"], "unicode_escape"), msg=evts[0])
