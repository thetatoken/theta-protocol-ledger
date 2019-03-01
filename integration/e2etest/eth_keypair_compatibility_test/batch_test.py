import os
import re
import sys
import subprocess

#
# Configs
#
ETHEREUM_ROOT = '~/.ethereum'
THETACLI_ROOT = '~/.thetacli'
NEW_ACCOUNT_PASSWORD_FILEPATH = './new_account_password.txt'
FAUCET_ADDRESS = '0x9f1233798e905e173560071255140b4a8abd3ec6'
FAUCET_PASSWORD = 'qwertyuiop'
UNLOCK_KEY_CMD_TMPL = """curl -X POST -H 'Content-Type: application/json' --data '{"jsonrpc":"2.0","method":"thetacli.UnlockKey","params":[{"address":"%s", "password":"%s"}],"id":1}' http://localhost:16889/rpc"""
SEND_CMD_TMPL = """curl -X POST -H 'Content-Type: application/json' --data '{"jsonrpc":"2.0","method":"thetacli.Send","params":[{"chain_id":"testnet", "from":"%s", "to":"%s", "thetawei":"%s", "tfuelwei":"%s", "fee":"1000000000000", "sequence":"%s", "async":false}],"id":1}' --silent --output /dev/null http://localhost:16889/rpc"""

def GenerateNewKeystore():
  geth_cmd = 'geth account new --datadir "%s" --password %s'%(ETHEREUM_ROOT, NEW_ACCOUNT_PASSWORD_FILEPATH)
  proc = subprocess.Popen([geth_cmd], stdout=subprocess.PIPE, shell=True)
  (out, err) = proc.communicate()
  if err != None:
    print("[ERROR] failed to execute cmd: %s"%(geth_cmd))
    exit(1)
  regex = re.compile("Address: {(?P<name>[0-9a-f]*)}")
  match = regex.match(out)
  if match == None or len(match.groups()) != 1:
    print("[ERROR] failed to extract address from: %s"%(out))
    exit(1)
  address = match.groups()[0]

  cp_cmd = 'cp %s/keystore/*--%s %s/keys/encrypted/%s'%(ETHEREUM_ROOT, address, THETACLI_ROOT, address)
  os.system(cp_cmd)
  
  return address

def GetInitialFaucetSeq():
  query_cmd = 'thetacli query account --address=%s'%(FAUCET_ADDRESS)
  proc = subprocess.Popen([query_cmd], stdout=subprocess.PIPE, shell=True)
  (out, err) = proc.communicate()
  if err != None:
    print("[ERROR] failed to execute cmd: %s"%(geth_cmd))
    exit(1)
  regex = re.compile('"sequence": "(?P<name>[0-9]*)"')
  match = regex.search(out)
  if match == None or len(match.groups()) != 1:
    print("[ERROR] failed to extract sequence from: %s"%(out))
    exit(1)
  sequence = int(match.groups()[0])
  init_faucet_seq = sequence + 1
  return init_faucet_seq

def BatchTest(init_faucet_seq):
  password_file = open(NEW_ACCOUNT_PASSWORD_FILEPATH, 'r')
  new_account_password = password_file.read().replace('\n', '')
  password_file.close()

  print("Unlock the faucet...")
  unlock_faucet_cmd = UNLOCK_KEY_CMD_TMPL%(FAUCET_ADDRESS, FAUCET_PASSWORD)
  os.system(unlock_faucet_cmd)
  print("Faucet unlocked.")
  print("")

  faucet_seq = init_faucet_seq
  for idx in range(500):
    print("----------------------------------------------------------------")
    print(">>>> TEST %s"%(idx))

    address = GenerateNewKeystore()
    
    print("Transfer some tokens from the faucet to 0x%s..."%(address))
    transfer_from_faucet_cmd = SEND_CMD_TMPL%(FAUCET_ADDRESS, address, 1000, 1000000000000000000, faucet_seq)
    os.system(transfer_from_faucet_cmd)
    faucet_seq = faucet_seq + 1
    print("Faucet transfer succeeded.")
    
    print("Unlocking address 0x%s..."%(address))
    unlock_address_cmd = UNLOCK_KEY_CMD_TMPL%(address, new_account_password)
    os.system(unlock_address_cmd)

    print("Transfer a portion of tokens back to the faucet from 0x%s..."%(address))
    transfer_back_to_faucet_cmd = SEND_CMD_TMPL%(address, FAUCET_ADDRESS, 19, 19, 1)
    #print(transfer_back_to_faucet_cmd)
    os.system(transfer_back_to_faucet_cmd)

    print("Transfer test completed for 0x%s"%(address))
    os.system("thetacli query account --address=%s"%(address))
    print("----------------------------------------------------------------")
    print("")

#
# __MAIN__
#
# Before running this script, we need to launch both the `theta` and `thetacli` daemon connected 
# to the testnet, and running  at port 16888 and 16889 respectively. Also need to install Geth
# to generate new ethereum accounts
#
if __name__ == '__main__':
  if len(sys.argv) != 1:
    print('\nUsage: python batch_test.py\n')
    exit(1)
  
  init_faucet_seq = GetInitialFaucetSeq()
  BatchTest(init_faucet_seq)

