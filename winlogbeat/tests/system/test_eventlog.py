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

    def write_event_log(self, message, eventID, sid=None):
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
            self.sidString = win32security.ConvertSidToStringSid(
                self.get_sid())

        return self.sidString

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
        self.read_one_event("wineventlog")

    def read_one_event(self, api):
        msg = "Read One Event Testcase"
        eventID = 11
        self.write_event_log(msg, eventID)

        # Run Winlogbeat
        self.render_config_template(
            event_logs=[
                {"name": self.providerName, "api": api}
            ]
        )
        proc = self.start_beat()
        self.wait_until(lambda: self.output_has(1))
        proc.kill()

        # Verify output
        events = self.read_output()
        assert len(events) == 1
        evt = events[0]
        assert evt["type"] == api
        assert evt["event_id"] == eventID
        assert evt["level"] == "Information"
        assert evt["log_name"] == self.providerName
        assert evt["source_name"] == self.applicationName
        assert evt["computer_name"].lower(
        ) == win32api.GetComputerName().lower()
        assert evt["user.identifier"] == self.get_sid_string()
        assert evt["user.name"] == win32api.GetUserName()
        assert "user.type" in evt
        assert "user.domain" in evt
        assert evt["message"] == msg

        exit_code = proc.wait()
        assert exit_code == 0

        return evt

    @unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
    def test_eventlogging_read_unknown_event_id(self):
        """
        Event Logging - Read unknown event ID
        """
        evt = self.read_unknown_event_id("eventlogging")

        assert "message_inserts" in evt
        assert evt["message_error"].lower() == ("The system cannot find "
                                                "message text for message number 1111 in the message file for "
                                                "C:\\Windows\\system32\\EventCreate.exe.").lower()

    @unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
    def test_wineventlog_read_unknown_event_id(self):
        """
        Win Event Log - Read unknown event ID
        """
        evt = self.read_unknown_event_id("wineventlog")

        # TODO: messageInserts has not been implemented for wineventlog.
        # assert "messageInserts" in evt
        assert evt["message_error"] == ("the message resource is present but "
                                        "the message is not found in the string/message table")

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
        proc.kill()

        # Verify output
        events = self.read_output()
        assert len(events) == 1
        evt = events[0]
        assert evt["type"] == api
        assert evt["event_id"] == eventID
        assert evt["level"] == "Information"
        assert evt["log_name"] == self.providerName
        assert evt["source_name"] == self.applicationName
        assert evt["computer_name"].lower(
        ) == win32api.GetComputerName().lower()
        assert evt["user.identifier"] == self.get_sid_string()
        assert evt["user.name"] == win32api.GetUserName()
        assert "user.type" in evt
        assert "user.domain" in evt
        assert "message" not in evt

        exit_code = proc.wait()
        assert exit_code == 0

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
        self.read_unknown_sid("wineventlog")

    def read_unknown_sid(self, api):
        # Fake SID that was made up.
        accountIdentifier = "S-1-5-21-3623811015-3361044348-30300820-1013"
        sid = win32security.ConvertStringSidToSid(accountIdentifier)

        msg = "Unknown SID of " + accountIdentifier
        eventID = 40
        self.write_event_log(msg, eventID, sid)

        # Run Winlogbeat
        self.render_config_template(
            event_logs=[
                {"name": self.providerName, "api": api}
            ]
        )
        proc = self.start_beat()
        self.wait_until(lambda: self.output_has(1))
        proc.kill()

        # Verify output
        events = self.read_output()
        assert len(events) == 1
        evt = events[0]
        assert evt["type"] == api
        assert evt["event_id"] == eventID
        assert evt["level"] == "Information"
        assert evt["log_name"] == self.providerName
        assert evt["source_name"] == self.applicationName
        assert evt["computer_name"].lower(
        ) == win32api.GetComputerName().lower()
        assert evt["user.identifier"] == accountIdentifier
        assert "user.name" not in evt
        assert "user.type" not in evt
        assert "user.domain" not in evt
        assert evt["message"] == msg

        exit_code = proc.wait()
        assert exit_code == 0

        return evt
