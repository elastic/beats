import hashlib
import os
import platform
import sys
import time
import yaml

if sys.platform.startswith("win"):
    import win32api
    import win32con
    import win32evtlog
    import win32security
    import win32evtlogutil

from beat.beat import TestCase

PROVIDER = "WinlogbeatTestPython"
APP_NAME = "SystemTest"
OTHER_APP_NAME = "OtherSystemTestApp"


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "winlogbeat"
        self.beat_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../"))
        super(BaseTest, self).setUpClass()


class WriteReadTest(BaseTest):
    providerName = PROVIDER
    applicationName = APP_NAME
    otherAppName = OTHER_APP_NAME
    testSuffix = None
    sid = None
    sidString = None
    api = None

    def setUp(self):
        super(WriteReadTest, self).setUp()

        # Every test will use its own event log and application names to ensure
        # isolation.
        self.testSuffix = "_" + hashlib.sha256(str(self.api + self._testMethodName).encode('utf_8')).hexdigest()[:5]
        self.providerName = PROVIDER + self.testSuffix
        self.applicationName = APP_NAME + self.testSuffix
        self.otherAppName = OTHER_APP_NAME + self.testSuffix

        win32evtlogutil.AddSourceToRegistry(self.applicationName,
                                            "%systemroot%\\system32\\EventCreate.exe",
                                            self.providerName)
        win32evtlogutil.AddSourceToRegistry(self.otherAppName,
                                            "%systemroot%\\system32\\EventCreate.exe",
                                            self.providerName)

    def tearDown(self):
        super(WriteReadTest, self).tearDown()
        self.clear_event_log()
        win32evtlogutil.RemoveSourceFromRegistry(
            self.applicationName, self.providerName)
        win32evtlogutil.RemoveSourceFromRegistry(
            self.otherAppName, self.providerName)

    def clear_event_log(self):
        hlog = win32evtlog.OpenEventLog(None, self.providerName)
        win32evtlog.ClearEventLog(hlog, None)
        win32evtlog.CloseEventLog(hlog)

    def write_event_log(self, message, eventID=10, sid=None,
                        level=None, source=None):
        if sid is None:
            sid = self.get_sid()
        if source is None:
            source = self.applicationName
        if level is None:
            level = win32evtlog.EVENTLOG_INFORMATION_TYPE

        # Retry on exception for up to 10 sec.
        t = time.monotonic()
        while True:
            try:
                win32evtlogutil.ReportEvent(source, eventID,
                                            eventType=level, strings=[message], sid=sid)
                break
            except:
                if time.monotonic() - t < 10:
                    continue
                raise

    def get_sid(self):
        if self.sid is None:
            ph = win32api.GetCurrentProcess()
            th = win32security.OpenProcessToken(ph, win32con.TOKEN_READ)
            self.sid = win32security.GetTokenInformation(
                th, win32security.TokenUser)[0]

        return self.sid

    def get_sid_string(self):
        if self.sidString is None:
            self.sidString = win32security.ConvertSidToStringSid(self.get_sid())

        return self.sidString

    def read_events(self, config=None, expected_events=1):
        if config is None:
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

    def read_registry(self, requireBookmark=False):
        f = open(os.path.join(self.working_dir, "data", ".winlogbeat.yml"), "r")
        data = yaml.load(f, Loader=yaml.FullLoader)
        self.assertIn("update_time", data)
        self.assertIn("event_logs", data)

        event_logs = {}
        for event_log in data["event_logs"]:
            self.assertIn("name", event_log)
            self.assertIn("record_number", event_log)
            self.assertIn("timestamp", event_log)
            if requireBookmark:
                self.assertIn("bookmark", event_log)
            name = event_log["name"]
            event_logs[name] = event_log

        return event_logs

    def assert_common_fields(self, evt, msg=None, eventID=10, sid=None,
                             level="information", extra=None):

        assert host_name(evt["winlog.computer_name"]).lower() == host_name(platform.node()).lower()
        assert "winlog.record_id" in evt
        expected = {
            "winlog.event_id": eventID,
            "event.code": eventID,
            "log.level": level.lower(),
            "winlog.channel": self.providerName,
            "winlog.provider_name": self.applicationName,
            "winlog.api": self.api,
        }
        assert expected.items() <= evt.items()

        if msg is None:
            assert "message" not in evt
        else:
            self.assertEqual(evt["message"], msg)
            self.assertEqual(msg, evt.get("winlog.event_data.param1"))

        if sid is None:
            self.assertEqual(evt["winlog.user.identifier"], self.get_sid_string())
            self.assertEqual(evt["winlog.user.name"].lower(),
                             win32api.GetUserName().lower())
            self.assertEqual(evt["winlog.user.type"], "User")
            assert "winlog.user.domain" in evt
        else:
            self.assertEqual(evt["winlog.user.identifier"], sid)
            assert "winlog.user.name" not in evt
            assert "winlog.user.type" not in evt

        if extra is not None:
            assert extra.items() <= evt.items()


def host_name(fqdn):
    return fqdn.split('.')[0]
