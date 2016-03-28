import sys

if sys.platform.startswith("win"):
    import win32api
    import win32con
    import win32evtlog
    import win32security
    import win32evtlogutil

sys.path.append('../../../libbeat/tests/system')
from beat.beat import TestCase


class BaseTest(TestCase):
    @classmethod
    def setUpClass(self):
        self.beat_name = "winlogbeat"
        super(BaseTest, self).setUpClass()

class WriteReadTest(BaseTest):
    providerName = "WinlogbeatTestPython"
    applicationName = "SystemTest"
    otherAppName = "OtherSystemTestApp"
    sid = None
    sidString = None
    api = None

    def setUp(self):
        super(WriteReadTest, self).setUp()
        win32evtlogutil.AddSourceToRegistry(self.applicationName,
                                            "%systemroot%\\system32\\EventCreate.exe",
                                            self.providerName)
        win32evtlogutil.AddSourceToRegistry(self.otherAppName,
                                            "%systemroot%\\system32\\EventCreate.exe",
                                            self.providerName)

    def tearDown(self):
        super(WriteReadTest, self).tearDown()
        win32evtlogutil.RemoveSourceFromRegistry(
            self.applicationName, self.providerName)
        win32evtlogutil.RemoveSourceFromRegistry(
            self.otherAppName, self.providerName)
        self.clear_event_log()

    def clear_event_log(self):
        hlog = win32evtlog.OpenEventLog(None, self.providerName)
        win32evtlog.ClearEventLog(hlog, None)
        win32evtlog.CloseEventLog(hlog)

    def write_event_log(self, message, eventID=10, sid=None,
                        level=None, source=None):
        if sid == None:
            sid = self.get_sid()
        if source == None:
            source = self.applicationName
        if level == None:
            level = win32evtlog.EVENTLOG_INFORMATION_TYPE

        win32evtlogutil.ReportEvent(source, eventID,
                                    eventType=level, strings=[message], sid=sid)

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

    def read_events(self, config=None, expected_events=1):
        if config == None:
            config = {
                "event_logs": [
                    {"name": self.providerName, "api": self.api}
                ]
            }

        self.render_config_template(**config)
        proc = self.start_beat()
        self.wait_until(lambda: self.output_has(expected_events))
        proc.check_kill_and_wait()

        return self.read_output()

    def assert_common_fields(self, evt, msg=None, eventID=10, sid=None,
                             level="Information", extra=None):
        assert evt["computer_name"].lower() == win32api.GetComputerName().lower()
        assert "record_number" in evt
        self.assertDictContainsSubset({
            "event_id": eventID,
            "level": level,
            "log_name": self.providerName,
            "source_name": self.applicationName,
            "type": self.api,
        }, evt)

        if msg == None:
            assert "message" not in evt
        else:
            self.assertEquals(evt["message"], msg)
            self.assertDictContainsSubset({"event_data.param1": msg}, evt)

        if sid == None:
            self.assertEquals(evt["user.identifier"], self.get_sid_string())
            self.assertEquals(evt["user.name"].lower(),
                              win32api.GetUserName().lower())
            self.assertEquals(evt["user.type"], "User")
            assert "user.domain" in evt
        else:
            self.assertEquals(evt["user.identifier"], sid)
            assert "user.name" not in evt
            assert "user.type" not in evt

        if extra != None:
            self.assertDictContainsSubset(extra, evt)
