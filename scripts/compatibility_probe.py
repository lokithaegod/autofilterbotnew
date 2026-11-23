#!/usr/bin/env python3
"""Read-only, fail-closed SRT-to-local-video analyzer. Emits a single JSON result."""
import argparse
import json
import os
import re
import shutil
import statistics
import subprocess
import tempfile
from pathlib import Path

STAMP=re.compile(r"(\d{2}):(\d{2}):(\d{2})[,\.](\d{3})\s*-->\s*(\d{2}):(\d{2}):(\d{2})[,\.](\d{3})")
SCORE=re.compile(r"(?im)^.*?score:\s*(-?\d+(?:\.\d+)?)")

def cues(path):
    data=Path(path).read_text(encoding="utf-8-sig",errors="replace")
    result=[]
    for match in STAMP.finditer(data):
        v=[int(n) for n in match.groups()]
        start=v[0]*3600+v[1]*60+v[2]+v[3]/1000
        end=v[4]*3600+v[5]*60+v[6]+v[7]/1000
        if end>start:result.append((start,end))
    return result

def compare_cues(before,after,quality_confirmed=False):
    if len(before)<12 or len(before)!=len(after):
        return {"status":"unverified","reason":"Missing or changed subtitle cues"}
    if before[-1][1]-before[0][0]<300:
        return {"status":"unverified","reason":"Subtitle spans less than five minutes"}
    shifts=[a[0]-b[0] for a,b in zip(before,after)]
    offset=statistics.median(shifts)
    width=max(3,len(shifts)//5)
    drift=statistics.median(shifts[-width:])-statistics.median(shifts[:width])
    max_shift=max(abs(x) for x in shifts)
    if max_shift<=.35 and abs(offset)<=.20 and abs(drift)<=.20:
        if quality_confirmed:
            kind="verified"
            reason="Audio speech-alignment score is positive; no meaningful shift required"
        else:
            kind="unverified"
            reason="Alignment quality was not independently established"
    elif abs(offset)<=120 and abs(drift)<=3:
        kind="needs_sync"
        reason="Audio analysis found offset or progressive timing drift"
    else:
        kind="unverified"
        reason="Alignment is too ambiguous to classify safely"
    return {"status":kind,"reason":reason,"offset_seconds":round(offset,3),
            "drift_seconds":round(drift,3),"confidence":"high" if kind=="verified" else "limited"}

def analyze(subtitle,video):
    if not Path(video).is_file():
        return {"status":"unverified","reason":"Video bytes are not accessible on analysis host"}
    if not Path(subtitle).is_file():
        return {"status":"unverified","reason":"Subtitle bytes unavailable"}
    original=cues(subtitle)
    if len(original)<12:return {"status":"unverified","reason":"Fewer than 12 valid SRT cues"}
    try:
        p=subprocess.run(["ffprobe","-v","error","-show_entries",
           "format=duration:stream=codec_type,avg_frame_rate","-of","json",video],
           capture_output=True,text=True,timeout=60,check=True)
        probe=json.loads(p.stdout)
        duration=float(probe.get("format",{}).get("duration",0))
        streams=probe.get("streams",[])
        video_stream=next((stream for stream in streams if stream.get("codec_type")=="video"),None)
        fps=None
        if video_stream:
            raw_rate=video_stream.get("avg_frame_rate","")
            try:
                numerator,denominator=(float(value) for value in raw_rate.split("/"))
                if denominator>0:fps=round(numerator/denominator,3)
            except (ValueError,ZeroDivisionError):pass
        metadata={"duration_seconds":round(duration,3),"fps":fps}
        if duration<=0 or not any(s.get("codec_type")=="video" for s in streams):
            raise ValueError("No valid video stream/duration")
        if original[-1][1]>duration+5:
            return {"status":"incompatible","reason":"Subtitle ends after the video runtime","video":metadata}
        if not any(s.get("codec_type")=="audio" for s in streams):
            return {"status":"unverified","reason":"Video has no audio stream","video":metadata}
    except (OSError,ValueError,subprocess.SubprocessError) as e:
        return {"status":"unverified","reason":"FFprobe unavailable or failed"}
    tool=shutil.which("ffsubsync") or shutil.which("ffs")
    if not tool:return {"status":"unverified","reason":"FFsubsync is not installed"}
    with tempfile.TemporaryDirectory(prefix="subtitle_compat_") as directory:
        out=os.path.join(directory,"aligned.srt")
        try:
            p=subprocess.run([tool,video,"-i",subtitle,"-o",out,
                "--skip-sync-on-low-quality","--min-score","0"],
                capture_output=True,text=True,timeout=180,check=False)
        except (OSError,subprocess.TimeoutExpired):
            return {"status":"unverified","reason":"Audio sync analysis timed out or failed"}
        if p.returncode!=0 or not os.path.isfile(out):
            return {"status":"unverified","reason":"FFsubsync did not generate alignment"}
        logs=p.stderr+"\n"+p.stdout
        matches=SCORE.findall(logs)
        reliable=bool(matches) and max(map(float,matches))>0 and "low-quality alignment" not in logs
        result=compare_cues(original,cues(out),quality_confirmed=reliable)
        result["video"]=metadata
        result["method"]="ffprobe and FFsubsync voice activity; original media preserved"
        return result

def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("--subtitle",required=True)
    parser.add_argument("--video",required=True)
    a=parser.parse_args()
    print(json.dumps(analyze(a.subtitle,a.video)))
if __name__=="__main__":main()
