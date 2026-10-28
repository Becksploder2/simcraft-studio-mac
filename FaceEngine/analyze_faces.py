"""MediaPipe reference/CAS facial landmarks and region metrics (no automatic CAS editing)."""
import argparse, json, sys, traceback
from pathlib import Path

REGIONS = {
 "eyes": [33,133,159,145,362,263,386,374],
 "eyebrows": [70,63,105,66,107,336,296,334,293,300],
 "nose": [1,4,5,6,98,327,168,197],
 "cheeks": [50,101,123,205,280,330,352,425],
 "mouth_lips": [61,291,13,14,0,17,78,308],
 "chin_jaw": [152,172,136,150,377,365,379,234,454],
 "forehead": [10,67,103,109,297,332,338],
}

def analyze(model_path, image_path):
 import mediapipe as mp
 from PIL import Image
 import numpy as np
 opts=mp.tasks.vision.FaceLandmarkerOptions(
  base_options=mp.tasks.BaseOptions(model_asset_path=str(model_path)),
  running_mode=mp.tasks.vision.RunningMode.IMAGE,
  num_faces=1, output_face_blendshapes=True)
 with mp.tasks.vision.FaceLandmarker.create_from_options(opts) as detector:
  # CAS screenshots often contain a relatively small face; also try a centered crop.
  source=Image.open(image_path).convert('RGB')
  candidates=[('full',source)]
  w,h=source.size
  if w>700 and h>500:
   candidates.append(('center_crop',source.crop((int(w*.2),int(h*.06),int(w*.8),int(h*.9)))))
  result=None; used='full'
  for name,picture in candidates:
   frame=mp.Image(image_format=mp.ImageFormat.SRGB,data=np.asarray(picture))
   trial=detector.detect(frame)
   if trial.face_landmarks:
    result=trial;used=name;break
 if result is None:
  return {'image':str(image_path),'face_count':0,'faces':[], 'note':'No face detected; use a clear, frontal close-up'}
 faces=[]
 for i,face in enumerate(result.face_landmarks):
  points=[{'x':round(pt.x,6),'y':round(pt.y,6),'z':round(pt.z,6)} for pt in face]
  regions={}
  for region,indices in REGIONS.items():
   pts=[face[j] for j in indices if j<len(face)]
   if pts:
    regions[region]={'center_x':round(sum(v.x for v in pts)/len(pts),5),
     'center_y':round(sum(v.y for v in pts)/len(pts),5),
     'width':round(max(v.x for v in pts)-min(v.x for v in pts),5),
     'height':round(max(v.y for v in pts)-min(v.y for v in pts),5)}
  shapes={v.category_name:round(v.score,5) for v in result.face_blendshapes[i]} if i<len(result.face_blendshapes) else {}
  faces.append({'landmarks':points,'regions':regions,'blendshapes':shapes})
 return {'image':str(image_path),'face_count':len(faces),'detected_in':used,'faces':faces}

def main():
 ap=argparse.ArgumentParser();ap.add_argument('photo',type=Path)
 ap.add_argument('--cas',type=Path)
 ap.add_argument('--model',type=Path,default=Path(__file__).resolve().parent.parent/'Models'/'face_landmarker.task')
 ap.add_argument('--output',type=Path,default=Path('face_analysis.json'))
 args=ap.parse_args()
 if not args.model.is_file():raise FileNotFoundError('Model missing: '+str(args.model))
 if not args.photo.is_file():raise FileNotFoundError('Photo missing: '+str(args.photo))
 data={'version':'4.3','reference':analyze(args.model,args.photo)}
 if args.cas is not None:
  if not args.cas.is_file():raise FileNotFoundError('CAS screenshot missing: '+str(args.cas))
  data['cas']=analyze(args.model,args.cas)
 args.output.parent.mkdir(parents=True,exist_ok=True)
 args.output.write_text(json.dumps(data,indent=2),encoding='utf-8')
 print('Analysis complete:',args.output)

if __name__=='__main__':
 try:main()
 except Exception:
  traceback.print_exc(file=sys.stderr)
  sys.exit(1)
