#!/usr/bin/env python3
"""Simcraft CC inventory: read-only DBPF package header scanner. Python 3.9+."""
import argparse, hashlib, json, os, pathlib, re, struct, sys, time
from collections import Counter

CATEGORIES = {
 'glasses':r'glasses|eyewear|spectacle|sunglass|frames',
 'hair':r'(^|[^a-z])hair([^a-z]|$)|hairstyle|bangs|ponytail|braid|bobcut',
 'shoes':r'shoes?|boots?|sneakers?|heels?',
 'top':r'(^|[^a-z])tops?([^a-z]|$)|shirt|blouse|hoodie|sweater|jacket|cardigan',
 'bottom':r'(^|[^a-z])bottoms?([^a-z]|$)|pants|jeans|skirts?|shorts|leggings',
 'fullbody':r'dress|jumpsuit|fullbody|full.body|outfit',
 'accessory':r'earrings?|necklace|bracelet|hat|ring|accessor',
 'makeup':r'makeup|lipstick|eyeshadow|eyeliner|blush',
 'skin':r'skin|freckle|overlay|tattoo',
}

def infer_category(name):
 s=name.lower().replace('_',' ').replace('-',' ')
 for cat, pat in CATEGORIES.items():
  if re.search(pat,s): return cat
 return 'unknown'

def read_header(path):
 with open(path,'rb') as f: h=f.read(96)
 if len(h)<32 or h[:4]!=b'DBPF':return {'valid_dbpf':False,'error':'Missing DBPF signature or truncated header'}
 major,minor=struct.unpack_from('<II',h,4)
 return {'valid_dbpf':True,'dbpf_major':major,'dbpf_minor':minor}

def scan(root,hash_files=False):
 root=pathlib.Path(root).expanduser().resolve()
 if not root.is_dir():raise ValueError('Mods folder does not exist: '+str(root))
 items=[];errors=[];duplicates={};total=0
 for base,dirs,files in os.walk(root):
  dirs[:]=[d for d in dirs if not pathlib.Path(base,d).is_symlink()]
  for filename in files:
   if not filename.lower().endswith('.package'):continue
   path=pathlib.Path(base,filename)
   if path.is_symlink():continue
   try:
    stat=path.stat();total+=stat.st_size
    header=read_header(path)
    relative=path.relative_to(root).as_posix()
    cat=infer_category(path.stem)
    item={'name':path.stem,'category':cat,'category_source':'filename_guess' if cat!='unknown' else 'unclassified', 'source':'cc','styles':[],'colors':[],'occasions':[],'file':relative,'size_bytes':stat.st_size,**header}
    if hash_files:
     digest=hashlib.sha256()
     with open(path,'rb') as f:
      for block in iter(lambda:f.read(1024*1024),b''):digest.update(block)
     item['sha256']=digest.hexdigest();duplicates.setdefault(item['sha256'],[]).append(relative)
    items.append(item)
   except (OSError,ValueError,struct.error) as e:errors.append({'file':str(path),'error':str(e)})
 items.sort(key=lambda x:x['file'].lower())
 summary={'package_count':len(items),'total_bytes':total,'valid_dbpf_count':sum(x['valid_dbpf'] for x in items),'categories_filename_inferred':dict(Counter(x['category'] for x in items)),'errors':len(errors),'duplicate_groups':sum(len(v)>1 for v in duplicates.values())}
 return {'schema':'simcraft-cc-inventory-v1','generated_utc':time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime()),'scan_root':str(root),'read_only':True,'summary':summary,'items':items,'errors':errors,'duplicate_sha256_groups':[v for v in duplicates.values() if len(v)>1]}

def main():
 p=argparse.ArgumentParser(description='Read-only Sims 4 .package inventory for Simcraft Studio')
 p.add_argument('mods_folder',nargs='?',help='Sims 4 Mods folder (defaults to Documents/Electronic Arts/The Sims 4/Mods)')
 p.add_argument('-o','--output',default='Simcraft-CC-Catalog.json')
 p.add_argument('--hash',action='store_true',help='Compute SHA-256 hashes to detect exact duplicates (slower)')
 args=p.parse_args()
 root=args.mods_folder or str(pathlib.Path.home()/'Documents'/'Electronic Arts'/'The Sims 4'/'Mods')
 try: result=scan(root,args.hash)
 except (OSError,ValueError) as e:p.error(str(e))
 with open(args.output,'w',encoding='utf-8') as f:json.dump(result,f,indent=2,ensure_ascii=False)
 print('Scanned:',result['summary']['package_count'],'packages; valid DBPF:',result['summary']['valid_dbpf_count'])
 print('Catalog:',os.path.abspath(args.output))
 print('Read-only scan; category guesses based on filenames, not verified CAS resources.')
 if result['errors']:print('Unreadable files:',len(result['errors']))
if __name__=='__main__':main()
