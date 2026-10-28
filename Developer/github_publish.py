"""Create signed manifest for upload as a GitHub Release asset.
Requires cryptography. Never upload the private key.
"""
import argparse, base64, hashlib, json
from pathlib import Path
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

parser=argparse.ArgumentParser()
parser.add_argument('--new-key',action='store_true')
parser.add_argument('--key',default='publisher-private.pem')
parser.add_argument('--version')
parser.add_argument('--notes',default='Simcraft update')
parser.add_argument('--owner')
parser.add_argument('--repo')
parser.add_argument('--windows')
parser.add_argument('--mac')
a=parser.parse_args()
keyfile=Path(a.key)
if a.new_key:
    if keyfile.exists():parser.error('Refusing to overwrite an existing private key')
    key=Ed25519PrivateKey.generate()
    keyfile.write_bytes(key.private_bytes(serialization.Encoding.PEM,serialization.PrivateFormat.PKCS8,serialization.NoEncryption()))
    keyfile.chmod(0o600)
    print('PUBLIC KEY:',base64.b64encode(key.public_key().public_bytes(serialization.Encoding.Raw,serialization.PublicFormat.Raw)).decode())
    print('Store the PRIVATE key securely, outside your public GitHub repository.')
else:
    if not all((a.version,a.owner,a.repo,a.windows)):parser.error('--version --owner --repo --windows required')
    key=serialization.load_pem_private_key(keyfile.read_bytes(),None)
    def artifact(filename):
        if not filename:return {'url':'','sha256':'','size':0}
        file=Path(filename)
        return {'url':f'https://github.com/{a.owner}/{a.repo}/releases/download/v{a.version}/{file.name}',
                'sha256':hashlib.sha256(file.read_bytes()).hexdigest(),'size':file.stat().st_size}
    payload={'app':'simcraft','version':a.version,'notes':a.notes,
             'windows':artifact(a.windows),'mac':artifact(a.mac)}
    raw=json.dumps(payload,separators=(',',':'),ensure_ascii=False).encode('utf-8')
    signed=b'{"payload":'+raw+b',"signature":'+json.dumps(base64.b64encode(key.sign(raw)).decode()).encode()+b'}'
    Path('simcraft-release.json').write_bytes(signed)
    print('Created simcraft-release.json. Upload alongside the update archives to GitHub Release v'+a.version)
