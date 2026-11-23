import pathlib
import tempfile
import unittest
from compatibility_probe import compare_cues, cues, analyze

def timeline(offset=0, drift=0):
    before=[(i*12.5,i*12.5+2) for i in range(60)]
    after=[(a-offset-drift*i/59,b-offset-drift*i/59) for i,(a,b) in enumerate(before)]
    return before,after

class TestCompatibility(unittest.TestCase):
    def test_aligned_with_quality(self):
        self.assertEqual(compare_cues(*timeline(),quality_confirmed=True)["status"],"verified")
    def test_no_quality_is_never_verified(self):
        self.assertEqual(compare_cues(*timeline())["status"],"unverified")
    def test_fixed_offset(self):
        self.assertEqual(compare_cues(*timeline(3.5))["status"],"needs_sync")
    def test_progressive_drift(self):
        self.assertEqual(compare_cues(*timeline(0,2))["status"],"needs_sync")
    def test_large_alignment_uncertain(self):
        self.assertEqual(compare_cues(*timeline(400))["status"],"unverified")
    def test_empty_video_not_verified(self):
        self.assertEqual(analyze("/not-found/any.srt","/not-found/any.mkv")["status"],"unverified")
    def test_srt_timestamps(self):
        with tempfile.TemporaryDirectory() as d:
            path=pathlib.Path(d)/"file.srt"
            path.write_text("1\\n00:00:01,100 --> 00:00:02,000\\nHello\\n".replace("\\n","\n"))
            self.assertEqual(cues(path),[(1.1,2)])
if __name__=="__main__":unittest.main()
