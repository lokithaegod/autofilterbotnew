import datetime as dt
import io
import os
import unittest
from contextlib import redirect_stdout
from unittest.mock import patch
from zoneinfo import ZoneInfo

from night_schedule import desired_online, find_our_monitor, reconcile, main

IST=ZoneInfo("Asia/Kolkata")

class TestSchedule(unittest.TestCase):
    def test_ist_boundaries(self):
        for hour,minute,expected in [(0,59,True),(1,0,False),(6,59,False),(7,0,True),(23,59,True)]:
            with self.subTest(hour=hour,minute=minute):
                local=dt.datetime(2026,11,22,hour,minute,tzinfo=IST)
                self.assertEqual(desired_online(local.astimezone(dt.timezone.utc)),expected)

    def test_no_secrets_is_noop(self):
        with patch.dict(os.environ,{"RENDER_API_KEY":"","UPTIMEROBOT_API_KEY":""}):
            with patch("night_schedule.reconcile") as reconcile_mock:
                with redirect_stdout(io.StringIO()) as stream:
                    self.assertEqual(main(),0)
                reconcile_mock.assert_not_called()
                self.assertIn("Not activated",stream.getvalue())

    def test_monitor_match_must_be_unique(self):
        with patch("night_schedule.uptime_robot_call",return_value={"stat":"ok","monitors":[
            {"id":1,"url":"https://another-site.example/"}
        ]}):
            with self.assertRaisesRegex(RuntimeError,"exactly one"):
                find_our_monitor("dummy")

    def test_quiet_hours_pauses_before_suspend(self):
        calls=[]
        with patch("night_schedule.find_our_monitor",return_value={"id":123,"url":"https://autofilterbotnew-ioi4.onrender.com/","status":2}):
            with patch("night_schedule.set_monitor",side_effect=lambda *a:calls.append("monitor")):
                with patch("night_schedule.set_render",side_effect=lambda *a:calls.append("render")):
                    reconcile("render-dummy","uptime-dummy",False)
        self.assertEqual(calls,["monitor","render"])

    def test_daytime_resumes_before_unpausing(self):
        calls=[]
        with patch("night_schedule.find_our_monitor",return_value={"id":123,"url":"https://autofilterbotnew-ioi4.onrender.com/","status":0}):
            with patch("night_schedule.set_render",side_effect=lambda *a:calls.append("render")):
                with patch("night_schedule.health_ready",return_value=True):
                    with patch("night_schedule.set_monitor",side_effect=lambda *a:calls.append("monitor")):
                        reconcile("render-dummy","uptime-dummy",True)
        self.assertEqual(calls,["render","monitor"])

if __name__=="__main__":
    unittest.main()
