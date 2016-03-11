import sys
import unittest
from winlogbeat import BaseTest

if sys.platform.startswith("win"):
    import win32api
    import win32con
    import win32evtlog
    import win32security
    import win32evtlogutil

"""
Contains tests for reading from the Windows Event Log (both APIs).
"""

class Test(BaseTest):
    providerName = "WinlogbeatTestPython"
    applicationName = "SystemTest"
    sid = None
    sidString = None

    def setUp(self):
        super(Test, self).setUp()
        win32evtlogutil.AddSourceToRegistry(self.applicationName,
                                            "%systemroot%\\system32\\EventCreate.exe",
                                            self.providerName)

    def tearDown(self):
        super(Test, self).tearDown()
        win32evtlogutil.RemoveSourceFromRegistry(
            self.applicationName, self.providerName)
        self.clear_event_log()

    def clear_event_log(self):
        hlog = win32evtlog.OpenEventLog(None, self.providerName)
        win32evtlog.ClearEventLog(hlog, None)
        win32evtlog.CloseEventLog(hlog)

    def write_event_log(self, message, eventID=10, sid=None):
        if sid == None:
            sid = self.get_sid()

        level = win32evtlog.EVENTLOG_INFORMATION_TYPE
        descr = [message]

        win32evtlogutil.ReportEvent(self.applicationName, eventID,
                                    eventType=level, strings=descr, sid=sid)

    def get_sid(self):
        if self.sid == None:
            ph = win32api.GetCurrentProcess()
            th = win32security.OpenProcessToken(ph, win32con.TOKEN_READ)
            self.sid = win32security.GetTokenInformation(
                th, win32security.TokenUser)[0]

        return self.sid

    def get_sid_string(self):
        if self.sidString == None:
            self.sidString = win32security.ConvertSidToStringSid(self.get_sid())

        return self.sidString

    def assert_common_fields(self, evt, api, msg=None, eventID=10, sid=None, extra=None):
        assert evt["computer_name"].lower() == win32api.GetComputerName().lower()
        assert "record_number" in evt
        self.assertDictContainsSubset({
            "count": 1,
            "event_id": eventID,
            "level": "Information",
            "log_name": self.providerName,
            "source_name": self.applicationName,
            "type": api,
        }, evt)

        if msg == None:
            assert "message" not in evt
        else:
            self.assertEquals(evt["message"], msg)
            self.assertDictContainsSubset({"event_data.param1": msg}, evt)

        if sid == None:
            self.assertEquals(evt["user.identifier"], self.get_sid_string())
            self.assertEquals(evt["user.name"].lower(), win32api.GetUserName().lower())
            self.assertEquals(evt["user.type"], "User")
            assert "user.domain" in evt
        else:
            self.assertEquals(evt["user.identifier"], sid)
            assert "user.name" not in evt
            assert "user.type" not in evt

        if extra != None:
            self.assertDictContainsSubset(extra, evt)

    @unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
    def test_eventlogging_read_one_event(self):
        """
        Event Logging - Read one event
        """
        self.read_one_event("eventlogging")

    @unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
    def test_wineventlog_read_one_event(self):
        """
        Win Event Log - Read one event
        """
        evt = self.read_one_event("wineventlog")
        self.assertDictContainsSubset({
            "keywords": ["Classic"],
            "opcode": "Info",
        }, evt)

    def read_one_event(self, api):
        msg = "Read One Event Testcase"
        self.write_event_log(msg)

        # Run Winlogbeat
        self.render_config_template(
            event_logs=[
                {"name": self.providerName, "api": api}
            ]
        )
        proc = self.start_beat()
        self.wait_until(lambda: self.output_has(1))
        proc.check_kill_and_wait()

        # Verify output
        events = self.read_output()
        assert len(events) == 1
        evt = events[0]
        self.assert_common_fields(evt, api, msg)
        return evt

    @unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
    def test_eventlogging_read_unknown_event_id(self):
        """
        Event Logging - Read unknown event ID
        """
        evt = self.read_unknown_event_id("eventlogging")

        assert evt["message_error"].lower() == ("The system cannot find "
                                                "message text for message number 1111 in the message file for "
                                                "C:\\Windows\\system32\\EventCreate.exe.").lower()

    @unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
    def test_wineventlog_read_unknown_event_id(self):
        """
        Win Event Log - Read unknown event ID
        """
        evt = self.read_unknown_event_id("wineventlog")
        self.assertDictContainsSubset({
            "keywords": ["Classic"],
            "opcode": "Info",
        }, evt)

        # No rendering error is being given.
        #assert evt["message_error"] == ("the message resource is present but "
        #                                "the message is not found in the string/message table")

    def read_unknown_event_id(self, api):
        msg = "Unknown Event ID Testcase"
        eventID = 1111
        self.write_event_log(msg, eventID)

        # Run Winlogbeat
        self.render_config_template(
            event_logs=[
                {"name": self.providerName, "api": api}
            ]
        )
        proc = self.start_beat()
        self.wait_until(lambda: self.output_has(1))
        proc.check_kill_and_wait()

        # Verify output
        events = self.read_output()
        assert len(events) == 1
        evt = events[0]
        self.assert_common_fields(evt, api, None, eventID=1111, extra={"event_data.param1": msg})
        return evt

    @unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
    def test_eventlogging_read_unknown_sid(self):
        """
        Event Logging - Read event with unknown SID
        """
        self.read_unknown_sid("eventlogging")

    @unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
    def test_wineventlog_read_unknown_sid(self):
        """
        Win Event Log - Read event with unknown SID
        """
        evt = self.read_unknown_sid("wineventlog")
        self.assertDictContainsSubset({
            "keywords": ["Classic"],
            "opcode": "Info",
        }, evt)

    def read_unknown_sid(self, api):
        # Fake SID that was made up.
        accountIdentifier = "S-1-5-21-3623811015-3361044348-30300820-1013"
        sid = win32security.ConvertStringSidToSid(accountIdentifier)

        msg = "Unknown SID of " + accountIdentifier
        self.write_event_log(msg, sid=sid)

        # Run Winlogbeat
        self.render_config_template(
            event_logs=[
                {"name": self.providerName, "api": api}
            ]
        )
        proc = self.start_beat()
        self.wait_until(lambda: self.output_has(1))
        proc.check_kill_and_wait()

        # Verify output
        events = self.read_output()
        assert len(events) == 1
        evt = events[0]
        self.assert_common_fields(evt, api, msg, sid=accountIdentifier)
        return evt

    @unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
    def test_eventlogging_fields_under_root(self):
        """
        Event Logging - Fields Under Root
        """
        self.fields_under_root("eventlogging")

    @unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
    def test_wineventlog_fields_under_root(self):
        """
        Win Event Log - Fields Under Root
        """
        self.fields_under_root("wineventlog")

    def fields_under_root(self, api):
        msg = "Add fields under root"
        self.write_event_log(msg)

        # Run Winlogbeat
        self.render_config_template(
            tags = ["global"],
            fields = {"global": "field", "env": "prod", "type": "overwrite"},
            fields_under_root = True,
            event_logs = [
                {"name": self.providerName,
                 "api": api,
                 "tags": ["local"],
                 "fields_under_root": True,
                 "fields": {"local": "field", "env": "dev"}}
            ]
        )
        proc = self.start_beat()
        self.wait_until(lambda: self.output_has(1))
        proc.check_kill_and_wait()

        # Verify output
        events = self.read_output()
        self.assertEqual(len(events), 1)
        evt = events[0]
        self.assertDictContainsSubset({
            "global": "field",
            "env": "dev",
            "type": "overwrite",
            "local": "field",
            "tags": ["global", "local"],
        }, evt)
        return evt

    @unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
    def test_eventlogging_fields_not_under_root(self):
        """
        Event Logging - Fields Not Under Root
        """
        self.fields_not_under_root("eventlogging")

    @unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
    def test_wineventlog_fields_not_under_root(self):
        """
        Win Event Log - Fields Not Under Root
        """
        self.fields_not_under_root("wineventlog")

    def fields_not_under_root(self, api):
        msg = "Add fields"
        self.write_event_log(msg)

        # Run Winlogbeat
        self.render_config_template(
            fields = {"global": "field", "env": "prod", "type": "overwrite"},
            event_logs = [
                {"name": self.providerName,
                 "api": api,
                 "fields": {"local": "field", "env": "dev", "num": 1}}
            ]
        )
        proc = self.start_beat()
        self.wait_until(lambda: self.output_has(1))
        proc.check_kill_and_wait()

        # Verify output
        events = self.read_output()
        self.assertEqual(len(events), 1)
        evt = events[0]
        assert "tags" not in evt, "tags present in event"
        self.assertDictContainsSubset({
            "fields.global": "field",
            "fields.env": "dev",
            "fields.type": "overwrite",
            "fields.local": "field",
            "fields.num": 1,
        }, evt)
        return evt
